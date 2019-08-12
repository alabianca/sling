# Sling

Share files in the local network using MDNS for peer discovery.

### Install
`cd` into the project directory and execute `make`. It will install the binary in the `~/go/bin` directory

### Share
`echo Hello World | sling -remote=alice` on one machine. Then on another machine in the same network run `sling -uid=alice > hello.txt`

Sling uses MDNS to discover peers in the local network. A sharing peer will read from `stdin` and a receiving peer will write to `stdin` allowing you to hook up the input/output to any other program you need. 

#### Command Options

| Command       | Use                            |
| --------------|:------------------------------:|
| -h            | display help text              |
| -uid          | your peer id (set to anything) |
| -p            | port to listen on              |
