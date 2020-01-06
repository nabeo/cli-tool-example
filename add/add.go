package add

import (
	"fmt"
	"net"

	"github.com/nabeo/cli-tool-example/utils"

	"github.com/urfave/cli/v2"
)

// Command cli.Command object list
var Command = cli.Command{
  Name: "add",
  Aliases: []string{"a"},
  Usage: "add command",
  Action: doAdd,
  Flags: []cli.Flag{
    &cli.StringFlag{
      Name: "hostname",
      Usage: "hostname",
      Required: true,
      Aliases: []string{"H"},
    },
    &cli.StringFlag{
      Name: "ip",
      Usage: "IP Address",
      Aliases: []string{"i"},
    },
    &cli.StringFlag{
      Name: "cname",
      Usage: "CNAME record",
      Aliases: []string{"c"},
    },
    &cli.StringFlag{
      Name: "type",
      Usage: "A or CNAME",
      Aliases: []string{"t"},
    },
    &cli.StringFlag{
      Name: "zone",
      Usage: "Hosted Zone name",
      Required: true,
      Aliases: []string{"z"},
    },
  },
}

type addData struct {
  hostname string
  ip net.IP
  cname string
  zonename string
  zoneID string
}

func doAdd(c *cli.Context) (err error) {
  if len(c.String("ip")) > 0 && len(c.String("cname")) > 0 {
    return fmt.Errorf("choose ip or cname")
  }

  var data addData
  data.hostname = c.String("hostname")
  
  data.ip = net.ParseIP(c.String("ip"))
  data.zonename = c.String("zone")

  awsClient, err := utils.NewAWSClient(c)
  if err != nil {
    return err
  }
  data.zoneID, err = awsClient.GetHostedZoneID(data.zonename)
  if err != nil {
    return err
  }

  fmt.Println("added: ", c.Args().First())
  return nil
}
