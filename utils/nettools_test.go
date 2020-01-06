package utils

import (
  "testing"

  "net"
)

func TestGenerateReverseRecord(t *testing.T) {
  patterns := []struct {
    ip net.IP
    expected string
  }{
    { net.IPv4(192, 168, 0, 1), "1.0.168.192.in-addr.arpa." },
  }

  for idx, pattern := range patterns {
    actual := GenerateReverseRecord(pattern.ip)
    if pattern.expected != actual {
      t.Errorf("pattern %d: want %s, actual %s", idx, pattern.expected, actual)
    }
  }
}
