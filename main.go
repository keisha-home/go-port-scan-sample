package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	help          bool
	targets       string
	portRanges    string
	numOfgoroutine int
	file          string
	timeoutMS     int
	outputPath    string
	outputFile    *os.File
)

// ----------------------------------------------------------------------
func usage() {
	fmt.Fprintf(os.Stderr, `Usage: PortScan [-h] [-t IP/CIDR] [-p Ports] [-n Threads] [-f File] [-o OutputFile] [-w Timeout]

Options:
`)
	flagSet := flag.CommandLine
	order := []string{"h", "t", "p", "n", "f", "o", "w"}
	for _, name := range order {
		fl4g := flagSet.Lookup(name)
		fmt.Printf(" -%-2s %s\n", fl4g.Name, fl4g.Usage)
	}
}

func init() {
	flag.BoolVar(&help, "h", false, "Print the help page")
	flag.StringVar(&file, "f", "", "Import ip lists file. Ex: ip.txt")
	flag.StringVar(&targets, "t", "", "Targets, Single IP or CIDR. Ex: 192.168.2.1 or 192.168.2.0/24")
	flag.StringVar(&portRanges, "p", "", "Port ranges. Ex: 1-65535 or 80,443 or 80,443,100-110")
	flag.IntVar(&numOfgoroutine, "n", 20, "Number of concurrent goroutines (default 20)")
	flag.StringVar(&outputPath, "o", "", "Output file to save open ports (append mode)")
	flag.IntVar(&timeoutMS, "w", 3000, "Connection timeout in milliseconds (default 3000ms)")
	flag.Usage = usage
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func CIDR2IP(cidr string) ([]string, error) {
	var hosts []string
	if !strings.ContainsAny(cidr, "/") {
		hosts = append(hosts, cidr)
		return hosts, nil
	}
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		hosts = append(hosts, ip.String())
	}

	if len(hosts) > 2 {
		return hosts[1 : len(hosts)-1], nil
	}
	return hosts, nil
}

func ParsePortRange(portList string) ([]int, error) {
	var ports []int
	portList2 := strings.Split(portList, ",")
	for _, i := range portList2 {
		if strings.Contains(i, "-") {
			a := strings.Split(i, "-")
			startPort, _ := strconv.Atoi(a[0])
			endPort, _ := strconv.Atoi(a[1])
			for j := startPort; j <= endPort; j++ {
				ports = append(ports, j)
			}
		} else {
			singlePort, _ := strconv.Atoi(i)
			ports = append(ports, singlePort)
		}
	}
	return ports, nil
}

func isOpen(target string) bool {
	host, _, errSplit := net.SplitHostPort(target)
	network := "tcp"
	if errSplit == nil {
		if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
			network = "tcp4"
		}
	}
	conn, err := net.DialTimeout(network, target, time.Millisecond*time.Duration(timeoutMS))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}


func reportOpen(target string) {
	msg := fmt.Sprintf("[+] %s is open!\n", target)
	fmt.Print(msg)

	if outputFile != nil {
		_, err := outputFile.WriteString(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to output file: %v\n", err)
		}
	}
}

func portScan(hosts []string, ports []int) {
	wg := sync.WaitGroup{}
	targetsChan := make(chan string, 100)
	poolCount := numOfgoroutine


	for i := 0; i < poolCount; i++ {
		go func() {
			for j := range targetsChan {
				if isOpen(j) {
					reportOpen(j)
				}
				wg.Done()
			}
		}()
	}


	for _, m := range ports {
		portString := strconv.Itoa(m)
		for _, n := range hosts {
			target := n + ":" + portString
			wg.Add(1)
			targetsChan <- target
		}
	}

	close(targetsChan)
	wg.Wait()
}

func ReadFile(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func main() {
	flag.Parse()

	if flag.NFlag() == 0 || help {
		flag.Usage()
		os.Exit(0)
	}


	if outputPath != "" {
		var err error
		outputFile, err = os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open output file %s: %v\n", outputPath, err)
			os.Exit(1)
		}
		defer outputFile.Close()
	}

	var hosts []string
	var err error

	if file != "" {
		hosts = ReadFile(file)
	} else if targets != "" {
		hosts, err = CIDR2IP(targets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid target/CIDR: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: must specify either -t or -f\n")
		flag.Usage()
		os.Exit(1)
	}

	if portRanges == "" {
		fmt.Fprintf(os.Stderr, "Error: -p is required\n")
		flag.Usage()
		os.Exit(1)
	}

	ports, err := ParsePortRange(portRanges)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid port range: %v\n", err)
		os.Exit(1)
	}

	portScan(hosts, ports)
	fmt.Println("Done!")
}
