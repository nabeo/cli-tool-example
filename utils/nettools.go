package utils

import (
  "net"
  "strings"
  "strconv"
)

// GenerateReverseRecord ...
func GenerateReverseRecord(ip net.IP) (reverseRecord string) {
  ipv4 := ip.To4()
  r1 := []string{
    strconv.Itoa(int(ipv4[3])),
    strconv.Itoa(int(ipv4[2])),
    strconv.Itoa(int(ipv4[1])),
    strconv.Itoa(int(ipv4[0])),
    "in-addr", "arpa"}
  r2 := strings.Join(r1[:], ".")
  reverseRecord = strings.Join([]string{r2, "."}, "")
  return reverseRecord
}
