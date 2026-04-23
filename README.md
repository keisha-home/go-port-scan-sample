# go-port-scan-sample
A simple Go port scanning tool, modified from another project.

# build
Just "go build main.go"

# usage
> main.exe
Usage: PortScan [-h] [-t IP/CIDR] [-p Ports] [-n Threads] [-f File] [-o OutputFile] [-w Timeout]
Options:
 -h  Print the help page
 -t  Targets, Single IP or CIDR. Ex: 192.168.2.1 or 192.168.2.0/24
 -p  Port ranges. Ex: 1-65535 or 80,443 or 80,443,100-110
 -n  Number of concurrent goroutines (default 20)
 -f  Import ip lists file. Ex: ip.txt
 -o  Output file to save open ports (append mode)
 -w  Connection timeout in milliseconds (default 3000ms)
