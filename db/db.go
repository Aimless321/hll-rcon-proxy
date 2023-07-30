package db

import (
	"github.com/gookit/config/v2"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"time"
)

type Repository struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
}

func NewRepository() *Repository {
	client := influxdb2.NewClient(config.String("influxdb.url"), config.String("influxdb.token"))
	writeAPI := client.WriteAPI(config.String("influxdb.org"), config.String("influxdb.bucket"))

	return &Repository{
		client:   client,
		writeAPI: writeAPI,
	}
}

func (r *Repository) WriteRCONPoint(serverName, cmd string, args string, numBytes int) {
	tags := map[string]string{
		"server": serverName,
		"cmd":    cmd,
		"args":   args,
	}
	fields := map[string]interface{}{
		"bytes": numBytes,
	}
	point := write.NewPoint("command", tags, fields, time.Now())
	r.writeAPI.WritePoint(point)
}

func (r *Repository) Close() {
	r.writeAPI.Flush()
	r.client.Close()
}
