package delete

import (
  "net"

	"github.com/nabeo/cli-tool-example/utils"
	"github.com/urfave/cli/v2"
)

// Command cli.Command object list
var Command = cli.Command{
  Name: "delete",
  Aliases: []string{"del"},
  Action: doDelete,
  Flags: []cli.Flag{
    &cli.StringFlag{
      Name: "hostname",
      Usage: "hostname",
      Required: true,
      Aliases: []string{"H"},
    },
    &cli.StringFlag{
      Name: "zone",
      Usage: "HostedZone Name",
      Required: true,
      Aliases: []string{"z"},
    },
  },
}

type delData struct {
  hostname string
  zoneName string
  zoneID string
}

func doDelete(c *cli.Context) (err error){
  var data delData
  data.hostname = c.String("hostname")
  data.zoneName = c.String("zone")

  awsClient, err := utils.NewAWSClient(c)
  if err != nil {
    return err
  }

  data.zoneID, err = awsClient.GetHostedZoneID(data.zoneName)
  if err != nil {
    return err
  }

  var confToml utils.ConfToml
  err = utils.LoadConf(c.String("conf"), &confToml)
  if err != nil {
    return err
  }

  var rInfos utils.ReverseHostedZoneInfos
  for _, p := range confToml.ReverseHostedZones {
    rInfo, err := awsClient.CreateReverseHostedZoneInfo(p.NetworkCIDR, p.ZoneName)
    if err != nil {
      return err
    }
    rInfos.ReverseHostedZoneInfo[len(rInfos.ReverseHostedZoneInfo)] = rInfo
  }

  rr, err := awsClient.GetResourceRecordSetByName(data.hostname, data.zoneName)
  if err != nil {
    return err
  }

  switch *rr.Type {
  case "A":
    ip := net.ParseIP(*rr.ResourceRecords[0].Value)
    err = awsClient.RemoveAResourceRecordSet(ip, data.hostname, data.zoneName, rInfos)
    if err != nil {
      return err
    }
  case "CNAME":
    err = awsClient.RemoveCnameResourceRecordSet(data.hostname, *rr.ResourceRecords[0].Value, data.zoneName)
    if err != nil {
      return err
    }
  }

  return nil
}
