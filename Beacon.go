package main
 
import (
    "os"
    "fmt"
    "net"
    "time"
    "strconv"
    "math/rand"
    "encoding/json"

    "github.com/op/go-logging"
)


// +++++++++++++++++++++++++++
// +++++++++ Go-Logging Conf
// +++++++++++++++++++++++++++
var log = logging.MustGetLogger("treesip")

var format = logging.MustStringFormatter(
    "%{level:.4s}** %{time:0102 15:04:05.999999} %{pid} %{shortfile} %{message}",
)


// +++++++++ Constants
const (
    Port              = ":10001"
    Protocol          = "udp"
    BroadcastAddr     = "255.255.255.255"
)

// +++++++++ Global vars
var myIP net.IP = net.ParseIP("127.0.0.1")

// +++++++++ Channels
var buffer = make(chan string)
var done = make(chan bool)

// +++++++++ Packet structure
type Packet struct {
    Message      string     `json:"message"`
    Source       net.IP     `json:"source"`
}

 
// A Simple function to verify error
func CheckError(err error) {
    if err  != nil {
        log.Error("Error: ", err)
    }
}

// Getting my own IP, first we get all interfaces, then we iterate
// discard the loopback and get the IPv4 address, which should be the eth0
func SelfIP() net.IP {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        panic(err)
    }

    for _, a := range addrs {
        if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP
            }
        }
    }

    return net.ParseIP("127.0.0.1")
}

// Function that handles the buffer channel
func attendBufferChannel() {
    for {
        j, more := <-buffer
        if more {
            // s := strings.Split(j, "|")
            // _, jsonStr := s[0], s[1]

            // First we take the json, unmarshal it to an object
            packet := Packet{}
            json.Unmarshal([]byte(j), &packet)

            log.Info(myIP.String() + " -> Message: " + packet.Message + " from " + packet.Source.String())
        } else {
            fmt.Println("closing channel")
            done <- true
            return
        }
    }
}

func beacon() {
    ServerAddr,err := net.ResolveUDPAddr(Protocol, BroadcastAddr+Port)
    CheckError(err)
    LocalAddr, err := net.ResolveUDPAddr(Protocol, myIP.String()+":0")
    CheckError(err)
    Conn, err := net.DialUDP(Protocol, LocalAddr, ServerAddr)
    CheckError(err)
    defer Conn.Close()

    s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)
    t := strconv.Itoa(r1.Intn(100000))

    payload := Packet{
        Message: "Hello network! "+t,
        Source: myIP,
    }

    js, err := json.Marshal(payload)
    CheckError(err)

    log.Info("Our random message is "+t)

    if Conn != nil {
        msg := js
        buf := []byte(msg)
        for {
            _,err = Conn.Write(buf)
            CheckError(err)
            time.Sleep(time.Second * 5)
        }
    }
}
 
func main() {
    fmt.Printf("Hello World!")

    // +++++++++++++++++++++++++++++
    // ++++++++ Logger conf
    var logPath = "/var/log/golang/"
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        os.MkdirAll(logPath, 0777)
    }

    var logFile = logPath + "treesip.log"
    f, err := os.OpenFile(logFile, os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
    if err != nil {
        fmt.Printf("error opening file: %v", err)
    }

    // don't forget to close it
    defer f.Close()

    backend := logging.NewLogBackend(f, "", 0)
    backendFormatter := logging.NewBackendFormatter(backend, format)

    logging.SetBackend(backendFormatter)
    // ++++++++ END Logger conf
    // +++++++++++++++++++++++++++++

    log.Info("Starting UPD Beacon")

    // It gives one minute time for the network to get configured before it gets its own IP.
    time.Sleep(time.Second * 60)
    myIP = SelfIP();

    // Lets prepare a address at any address at port 10001
    ServerAddr,err := net.ResolveUDPAddr(Protocol, Port)
    CheckError(err)
 
    // Now listen at selected port
    ServerConn, err := net.ListenUDP(Protocol, ServerAddr)
    CheckError(err)
    defer ServerConn.Close()

    go attendBufferChannel()
    go beacon()
 
    buf := make([]byte, 1024)
 
    for {
        n,_,err := ServerConn.ReadFromUDP(buf)
        buffer <- string(buf[0:n])
        
        if err != nil {
            log.Error("Error: ",err)
        }
    }

    close(buffer)

    <-done
}