package utils

import (
  "github.com/BurntSushi/toml"
)

// ```
// [[ReverseHostedZone]]
// NetworkCIDR = "10.0.0.0/8"
// ZoneName = "10.in-addr.arpa."
// [[ReverseHostedZone]]
// NetworkCIDR = "192.168.0.0/16"
// ZoneName = "168.192.in-addr.arpa."
// [[ReverseHostedZone]]
// NetworkCIDR = "172.16.0.0/12"
// ZoneName = "16.172.in-addr.arpa."
// ```

// ConfToml ...
type ConfToml struct {
  ReverseHostedZones []ReverseHostedZone `toml:"ReverseHostedZone"`
}

// ReverseHostedZone ...
type ReverseHostedZone struct {
  NetworkCIDR string `toml:"NetworkCIDR"`
  ZoneName string `toml:"ZoneName"`
}

// LoadConf ...
func LoadConf(confPath string, confToml *ConfToml) (err error) {
  if _, err := toml.DecodeFile(confPath, confToml); err != nil {
    return err
  }

  return nil
}
