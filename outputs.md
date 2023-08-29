## Router outputs

### IOS XE

#### Configuration

```bash
# show run | section ip access-list extended IPv4-ACL
ip access-list extended IPv4-ACL
 10 deny tcp any 198.51.100.0 0.0.0.255
 20 permit tcp any any
```

#### Show command

```bash
# show ip access-lists IPv4-ACL
Extended IP access list IPv4-ACL
    10 deny tcp any 198.51.100.0 0.0.0.255
    20 permit tcp any any
```


### IOS XR

#### Configuration

```bash
# sh run ipv4 access-list IPv4-ACL
Tue Aug 22 19:28:23.909 UTC
ipv4 access-list IPv4-ACL
 10 deny tcp any 198.51.100.0 0.0.0.255
 20 permit tcp any any
!
```

#### Show command

```bash
# show access-lists IPv4-ACL 
Tue Aug 22 19:27:56.291 UTC
ipv4 access-list IPv4-ACL
 10 deny tcp any 198.51.100.0 0.0.0.255
 20 permit tcp any any


# show access-lists IPv4-ACL expanded 
Tue Aug 22 19:29:23.139 UTC
ipv4 access-list IPv4-ACL
 10 deny tcp any 198.51.100.0 0.0.0.255
 20 permit tcp any any
```