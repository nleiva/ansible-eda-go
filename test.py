import asyncio
import json
import logging
import shlex
from typing import Any, Dict

LOGGER = logging.getLogger("nleiva.eda.cmd")


async def process_output(
    proc: asyncio.subprocess.Process,
    queue: asyncio.Queue,
    deserialize: bool,
    command: str,
):
    # type hint warn: proc.stdout can be None, this is not our case
    while output := await proc.stdout.readline():
        try:
            if deserialize:
                event = json.loads(output.decode())
            else:
                event = output.decode()

            await queue.put({"cmd": event, "meta": {"command": command}})
        except json.JSONDecodeError as e:
            LOGGER.error("Can not deserialize JSON data: %s", e)


async def main(queue: asyncio.Queue, args: Dict[str, Any]):
    command = str(args["command"])
    repository = str(args["repository"])
    send_output = False
    stdout_mode = asyncio.subprocess.DEVNULL

    deserialize = bool(args.get("deserialize", True))
    send_output = bool(args.get("send_output", False))

    # Clone repo
    cloning = await asyncio.create_subprocess_shell('git clone --quiet ' + repository, stdout=stdout_mode)
    await cloning.wait()

    if send_output:
        stdout_mode = asyncio.subprocess.PIPE

    proc = await asyncio.subprocess.create_subprocess_shell(
        command,
        stdout=stdout_mode,
        stderr=asyncio.subprocess.PIPE,
    )

    LOGGER.info("Process started")

    if send_output:
        try:
            _unused_stdout, stderr = await asyncio.gather(
                process_output(proc, queue, deserialize, command),
                # type hint warn: proc.stderr can be None, this is not our case
                proc.stderr.read(),
            )
            await proc.wait()
        except asyncio.CancelledError:
            LOGGER.info("Ansible-rulebook is shutting down, stopping process")
            proc.terminate()
            _unused_stdout, stderr = await proc.communicate()
    else:
        _unused_stdout, stderr = await proc.communicate()

    LOGGER.info("Process finished")

    if proc.returncode != 0:
        LOGGER.error(
            "Command failed with code %s: %s", proc.returncode, stderr.decode()
        )

if __name__ == "__main__":
    """
    Entry point of the program.
    """

    class MockQueue:
        """A mock implementation of a queue that prints the event."""

        async def put(self: str, event: str) -> str:
            """Add the event to the queue and print it."""
            print(event)  # noqa: T201

    args = {
        "command": "cd ansible-eda-go && ./closed-loop",
        "repository": "https://github.com/nleiva/ansible-eda-go",
        "send_output": True,
        "deserialize": False,
    }

    asyncio.run(main(MockQueue(), args))