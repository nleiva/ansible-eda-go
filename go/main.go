package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"

	"gopkg.in/yaml.v2"
)

type Inventory struct {
	Routers []Router `yaml:"router"`
}

type DeviceInfo struct {
	Device    string
	Output    string
	Timestamp time.Time
}

type Event struct {
	Time    string `json:"time"`
	Device  string `json:"device"`
	Intent  string `json:"intent"`
	Current string `json:"current"`
	Failed  bool   `json:"failed"`
}

type Router struct {
	Hostname  string `yaml:"hostname"`
	Platform  string `yaml:"platform"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	StrictKey bool   `yaml:"strictkey"`
	Config    string `yaml:"config"`
	Check     string `yaml:"check"`
	Conn      *network.Driver
}

func (r Router) getOper(cmd string) (o DeviceInfo, err error) {
	rs, err := r.Conn.SendCommand(cmd)
	if err != nil {
		return o, fmt.Errorf("failed to send %s for %s: %w", cmd, r.Hostname, err)
	}
	o = DeviceInfo{
		Device:    r.Hostname,
		Output:    rs.Result,
		Timestamp: time.Now(),
	}
	return o, nil
}

func parseOper(input, platform string) (string, error) {
	if platform == "cisco_iosxr" {
		in := strings.NewReader(input)
		scanner := bufio.NewScanner(in)

		// Read first line
		scanner.Scan()
		oper := strings.TrimLeft(input, scanner.Text())

		return strings.TrimSpace(oper), nil
	}

	return input, nil
}

func check(err error, m string) {
	var e Event

	if err != nil {
		e.Current = fmt.Errorf("%s: %w", m, err).Error()
		e.Failed = true
		returnEvent(e)
	}
}

func returnEvent(e Event) {
	var response []byte

	if e.Time == "" {
		h, m, s := time.Now().Clock()
		e.Time = fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}

	response, err := json.Marshal(e)
	if err != nil {
		response, _ = json.Marshal(Event{
			Time:    e.Time,
			Current: fmt.Errorf("couldn't marshal event: %w", err).Error(),
			Failed:  true,
		})
	}

	fmt.Println(string(response))
	// if e.Failed {
	// 	os.Exit(1)
	// }
}

func main() {
	////////////////////////////////
	// Read input data
	////////////////////////////////
	timeout := 200

	src, err := os.Open("input.yml")
	check(err, "failed to open input data file")
	defer src.Close()

	d := yaml.NewDecoder(src)

	var inv Inventory
	err = d.Decode(&inv)
	check(err, "failed to parse input data file")

	////////////////////////////////
	// Continuous/Enforcement loop
	////////////////////////////////

	for _, device := range inv.Routers {
		go func(d Router) {
			////////////////////////////////////////
			// Gather network device info
			///////////////////////////////////////
			conn, err := platform.NewPlatform(
				d.Platform,
				d.Hostname,
				options.WithAuthNoStrictKey(),
				options.WithAuthUsername(d.Username),
				options.WithAuthPassword(d.Password),
				options.WithSSHConfigFile("ssh_config"),
			)
			check(err, "failed to create device platform for "+d.Platform)

			////////////////////////////////
			// Config verification
			////////////////////////////////
			cmd := d.Check

			////////////////////////////////
			// Compute hash of intended config
			////////////////////////////////
			intent, err := os.ReadFile(d.Config)
			check(err, "failed to read config for "+d.Platform)
			config := string(intent)

			intentHash, err := hashstructure.Hash(config,
				hashstructure.FormatV2, nil)
			check(err, "failed to compute config hash for "+d.Platform)

			////////////////////////////////////////
			// Open connection to the network device
			///////////////////////////////////////
			driver, err := conn.GetNetworkDriver()
			check(err, "failed to create device driver for "+d.Platform)

			d.Conn = driver
			err = driver.Open()
			check(err, "failed to open device driver for "+d.Platform)
			defer driver.Close()

			ticker := time.NewTicker(time.Second * 30)
			defer ticker.Stop()

			for ; true; <-ticker.C {
				////////////////////////////////
				// Get Operational Data
				////////////////////////////////
				opr, err := d.getOper(cmd)
				check(err, "failed to get operational status for "+d.Platform)

				////////////////////////////////
				// Parse Operational Data
				////////////////////////////////
				parsed, err := parseOper(opr.Output, d.Platform)
				check(err, "failed to parse operational status for "+d.Platform)

				////////////////////////////////
				// Validate State
				////////////////////////////////
				oprHash, err := hashstructure.Hash(parsed, hashstructure.FormatV2, nil)
				check(err, "failed to compute hash for operational status for "+d.Platform)

				h, m, s := time.Now().Clock()

				if oprHash == intentHash {
					// Match event
					event := Event{
						Time:    fmt.Sprintf("%02d:%02d:%02d", h, m, s),
						Device:  d.Hostname,
						Intent:  config,
						Current: parsed,
						Failed:  false,
					}

					returnEvent(event)
					continue

				}

				////////////////////////////////
				// Generate event (no hash match)
				////////////////////////////////
				event := Event{
					Time:    fmt.Sprintf("%02d:%02d:%02d", h, m, s),
					Device:  d.Hostname,
					Intent:  config,
					Current: parsed,
					Failed:  true,
				}

				returnEvent(event)

			}
		}(device)
	}

	// Program timeout
	time.Sleep(time.Duration(timeout) * time.Second)
	fmt.Println("End of the program")
}
