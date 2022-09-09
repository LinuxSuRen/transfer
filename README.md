[![codecov](https://codecov.io/gh/LinuxSuRen/transfer/branch/master/graph/badge.svg?token=XS8g2CjdNL)](https://codecov.io/gh/LinuxSuRen/transfer)

Data transfer via UDP protocol.

## Features
* Send files to an unknown target IP address in a local network
* Average speed is 15 MB/s in WiFi (802.11ac)

## Install
Using [hd](https://github.com/LinuxSuRen/http-downloader/) to install it:

```shell
hd i transfer
```

## Usage
Wait for the data:
```shell
transfer wait
```

send the data:
```shell
transfer send targetFile [ip]
```

## Limitations
* Not fast enough (8.35 MB/s) when sending data from macOS
