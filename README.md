## Example

```go
package main

import (
	"log/slog"
	"os"
	"time"

	"import.name/sjournal"
)

func main() {
	handler, err := sjournal.NewHandler(&sjournal.HandlerOptions{
		Delimiter:   sjournal.ColonDelimiter, // Not just space.
		IgnoreAttrs: []string{"request.header"},
		TimeFormat:  time.RFC3339Nano, // For attribute values.
	})
	if err != nil {
		slog.Error("journal initialization failed", "error", err)
		os.Exit(1)
	}

	logger := slog.New(handler)
	logger.Info("example message")
	logger.Info("another example message", "at", time.Now(), "foo", "bar baz")

	subsystemLogger := slog.New(handler.ExtendPrefix("zydeemi: "))
	subsystemLogger.Error("yet another example message")
}
```


### Short output

```
loka 05 14:39:51 tietokone example[242885]: example message
loka 05 14:39:51 tietokone example[242885]: another example message: at=2024-10-05T14:39:51.931753895+03:00 foo="bar baz"
loka 05 14:39:51 tietokone example[242885]: zydeemi: yet another example message
```

Error messages are red in journalctl output.


### Verbose output

```
Sat 2024-10-05 14:39:51.931748 EEST [s=0b338bada8cc43eab50388def922317c;i=131430b;b=8f9ef9fbef2f410898ef6d2565c164fd;m=1ddc7740a2;t=623b93eed6780;x=aaa8b78ab8489043]
    PRIORITY=6
    _TRANSPORT=journal
    _UID=1000
    _GID=1000
    _CAP_EFFECTIVE=0
    _SELINUX_CONTEXT=unconfined
    _AUDIT_SESSION=4
    _AUDIT_LOGINUID=1000
    _SYSTEMD_OWNER_UID=1000
    _SYSTEMD_UNIT=user@1000.service
    _SYSTEMD_SLICE=user-1000.slice
    _BOOT_ID=8f9ef9fbef2f410898ef6d2565c164fd
    _MACHINE_ID=c64667a6744d44da922acb9f4973ef62
    _HOSTNAME=tietokone
    _RUNTIME_SCOPE=system
    _SYSTEMD_USER_SLICE=app-org.gnome.Terminal.slice
    _SYSTEMD_CGROUP=/user.slice/user-1000.slice/user@1000.service/app.slice/app-org.gnome.Terminal.slice/vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_USER_UNIT=vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_INVOCATION_ID=c20a3b7c2f9248e29700ac2151046a89
    CODE_FILE=/home/user/sjournal-example/main.go
    CODE_LINE=25
    CODE_FUNC=main.main
    _COMM=example
    MESSAGE=example message
    SYSLOG_TIMESTAMP=1728128391
    _PID=242885
    _SOURCE_REALTIME_TIMESTAMP=1728128391931748
Sat 2024-10-05 14:39:51.931842 EEST [s=0b338bada8cc43eab50388def922317c;i=131430c;b=8f9ef9fbef2f410898ef6d2565c164fd;m=1ddc775173;t=623b93eed7850;x=7f7d0d632d83b03e]
    PRIORITY=6
    _TRANSPORT=journal
    _UID=1000
    _GID=1000
    _CAP_EFFECTIVE=0
    _SELINUX_CONTEXT=unconfined
    _AUDIT_SESSION=4
    _AUDIT_LOGINUID=1000
    _SYSTEMD_OWNER_UID=1000
    _SYSTEMD_UNIT=user@1000.service
    _SYSTEMD_SLICE=user-1000.slice
    _BOOT_ID=8f9ef9fbef2f410898ef6d2565c164fd
    _MACHINE_ID=c64667a6744d44da922acb9f4973ef62
    _HOSTNAME=tietokone
    _RUNTIME_SCOPE=system
    _SYSTEMD_USER_SLICE=app-org.gnome.Terminal.slice
    _SYSTEMD_CGROUP=/user.slice/user-1000.slice/user@1000.service/app.slice/app-org.gnome.Terminal.slice/vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_USER_UNIT=vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_INVOCATION_ID=c20a3b7c2f9248e29700ac2151046a89
    CODE_FILE=/home/user/sjournal-example/main.go
    CODE_FUNC=main.main
    _COMM=example
    CODE_LINE=26
    SYSLOG_TIMESTAMP=1728128391
    _PID=242885
    MESSAGE=another example message: at=2024-10-05T14:39:51.931753895+03:00 foo="bar baz"
    _SOURCE_REALTIME_TIMESTAMP=1728128391931842
Sat 2024-10-05 14:39:51.931853 EEST [s=0b338bada8cc43eab50388def922317c;i=131430d;b=8f9ef9fbef2f410898ef6d2565c164fd;m=1ddc7751be;t=623b93eed789b;x=2500cd27ec692b26]
    _TRANSPORT=journal
    _UID=1000
    _GID=1000
    _CAP_EFFECTIVE=0
    _SELINUX_CONTEXT=unconfined
    _AUDIT_SESSION=4
    _AUDIT_LOGINUID=1000
    _SYSTEMD_OWNER_UID=1000
    _SYSTEMD_UNIT=user@1000.service
    _SYSTEMD_SLICE=user-1000.slice
    _BOOT_ID=8f9ef9fbef2f410898ef6d2565c164fd
    _MACHINE_ID=c64667a6744d44da922acb9f4973ef62
    _HOSTNAME=tietokone
    _RUNTIME_SCOPE=system
    PRIORITY=3
    _SYSTEMD_USER_SLICE=app-org.gnome.Terminal.slice
    _SYSTEMD_CGROUP=/user.slice/user-1000.slice/user@1000.service/app.slice/app-org.gnome.Terminal.slice/vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_USER_UNIT=vte-spawn-94dc7a07-784b-4c6a-bc5a-12f0ed6d4d52.scope
    _SYSTEMD_INVOCATION_ID=c20a3b7c2f9248e29700ac2151046a89
    CODE_FILE=/home/user/sjournal-example/main.go
    CODE_FUNC=main.main
    _COMM=example
    MESSAGE=zydeemi: yet another example message
    CODE_LINE=29
    SYSLOG_TIMESTAMP=1728128391
    _PID=242885
    _SOURCE_REALTIME_TIMESTAMP=1728128391931853
```

The following fields were supplied by our Go program:

- CODE_FILE
- CODE_FUNC
- CODE_LINE
- MESSAGE
- PRIORITY
- SYSLOG_TIMESTAMP

