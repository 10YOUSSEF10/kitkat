package main

import (
    "bufio"
    "fmt"
    "github.com/fatih/color"
    "net"
    "net/http"
    "net/http/httptrace"
    "os"
    "strings"
    "sync"
    "time"
)

const maxResponseTime = 1

var transport = &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    DialContext: (&net.Dialer{
        Timeout:   5 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
}

var clientPool = sync.Pool{New: func() interface{} {
    return &http.Client{Timeout: 3 * time.Second, Transport: transport}
}}

func checkSite(site string, protocols []string) (string, float64) {
    client := clientPool.Get().(*http.Client)
    defer clientPool.Put(client)
    minElapsedTime := 9999.0

    for _, protocol := range protocols {
        url := site
        if !strings.HasPrefix(site, "http") {
            url = protocol + "://" + site
        }

        var start, firstByte time.Time
        trace := &httptrace.ClientTrace{GotFirstResponseByte: func() { firstByte = time.Now() }}
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            continue
        }

        req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
        start = time.Now()
        resp, err := client.Do(req)
        if err != nil {
            continue
        }
        resp.Body.Close()

        if !firstByte.IsZero() {
            elapsedTime := float64(firstByte.Sub(start).Milliseconds()) / 1000.0
            if elapsedTime < minElapsedTime {
                minElapsedTime = elapsedTime
            }
        }
    }

    if minElapsedTime < maxResponseTime {
        return site, minElapsedTime
    }
    return "", 0
}

func testSites(sites []string, protocols []string, maxWorkers int, outputFile string) {
    var mu sync.Mutex
    foundSites := make([]string, 0, len(sites))
    sem := make(chan struct{}, maxWorkers)
    progressChan := make(chan int, len(sites))

    go func() {
        completed := 0
        for range progressChan {
            completed++
            fmt.Printf("\rProgress: %d/%d sites checked", completed, len(sites))
        }
        fmt.Println()
    }()

    color.Cyan("\n🔄 Scanning %d sites...\n", len(sites))

    var wg sync.WaitGroup
    for _, site := range sites {
        wg.Add(1)
        sem <- struct{}{}
        go func(site string) {
            defer wg.Done()
            defer func() { <-sem }()
            result, elapsedTime := checkSite(site, protocols)
            progressChan <- 1
            if result != "" {
                color.Green("\n✔️ %s - %.2fs\n", result, elapsedTime)
                mu.Lock()
                foundSites = append(foundSites, result)
                mu.Unlock()
            }
        }(site)
    }

    wg.Wait()
    close(progressChan)

    if len(foundSites) == 0 {
        color.Yellow("\n⚠️ No sites responded within the expected time.\n")
    } else if outputFile != "" {
        file, err := os.Create(outputFile)
        if err != nil {
            fmt.Println("Error creating output file:", err)
            return
        }
        defer file.Close()
        for _, site := range foundSites {
            file.WriteString(strings.TrimPrefix(strings.TrimPrefix(site, "https://"), "http://") + "\n")
        }
        color.Blue("\n📁 Results saved to: %s\n", outputFile)
    }

    fmt.Println()
    color.Magenta("Channel: https://t.me/INTERNET_EGYPT_YOUSSEF")
    color.Magenta("Contact: @N0T_ROBOT\n")
}

func printUsage() {
    fmt.Println(`
Usage: go run gold.go [options]
Options:
  -h             Display this help message.
  -hs            Scan using HTTPS protocol.
  -o <file>      Save results to a file.
  -i <file>      Load sites from a file.
  -t <number>    Set the number of threads for concurrent scanning.
`)
}

func printKitKat() {
    red := color.New(color.FgRed).Add(color.Bold)
    red.Println(`
 K   K   III  TTTTT      K   K   AAAAA   TTTTT
 K  K     I     T        K  K   A     A    T
 KKK      I     T        KKK    AAAAAAA    T
 K  K     I     T        K  K   A     A    T
 K   K   III    T        K   K   A     A    T


channel: https://t.me/INTERNET_EGYPT_YOUSSEF

contact: @N0T_ROBOT
    `)
}

func main() {
    printKitKat()

    var outputFile, inputFile string
    maxWorkers, sites := 10, []string{}
    protocol := "https"
    args := os.Args[1:]

    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "-hs":
            protocol = "https"
        case "-o":
            outputFile, i = args[i+1], i+1
        case "-i":
            inputFile, i = args[i+1], i+1
        case "-t":
            fmt.Sscanf(args[i+1], "%d", &maxWorkers)
            i++
        default:
            printUsage()
            return
        }
    }

    if inputFile != "" {
        file, err := os.Open(inputFile)
        if err != nil {
            fmt.Printf("⚠️ File %s not found!\n", inputFile)
            return
        }
        defer file.Close()
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            if line := scanner.Text(); line != "" {
                sites = append(sites, line)
            }
        }
    }

    if len(sites) == 0 {
        fmt.Println("⚠️ No sites provided!")
        return
    }

    testSites(sites, []string{protocol}, maxWorkers, outputFile)
}
