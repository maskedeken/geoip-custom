package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	log "github.com/sirupsen/logrus"

	"github.com/EvilSuperstars/go-cidrman"

	"v2ray.com/core/app/router"
	"v2ray.com/core/infra/conf"

	"google.golang.org/protobuf/proto"
)

var (
	dataPath    string
	outputDir   string
	exportLists string
)

func init() {
	flag.StringVar(&dataPath, "datapath", "./data", "specify directory which contains ip list files")
	flag.StringVar(&outputDir, "outputdir", "./", "Directory to place all generated files")
	flag.StringVar(&exportLists, "exportlists", "", "Lists to be flattened and exported in plaintext format, separated by ',' comma")
	flag.Parse()
}

func load(path string) ([]*net.IPNet, error) {
	var ipList []*net.IPNet

	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		cidrTxt := scanner.Text()
		_, network, err := net.ParseCIDR(scanner.Text())
		if err != nil || network == nil {
			log.Printf("%s fail to parse to CIDR", cidrTxt)
			continue
		}
		ipList = append(ipList, network)
	}

	return ipList, nil
}

func main() {
	// Flatten and export plaintext list
	exportMap := make(map[string]interface{})
	if exportLists != "" {
		for _, name := range strings.Split(exportLists, ",") {
			exportMap[name] = nil
		}
	}

	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            "GeoIP2-Country",
			RecordSize:              24,
			IncludeReservedNetworks: true,
		},
	)
	if err != nil {
		log.Fatalf("fail to new writer: %v", err)
	}

	// Create output directory if not exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(outputDir, 0755); mkErr != nil {
			log.Fatalf("fail to create output directory: %v", err)
		}
	}

	ref := make(map[string][]*net.IPNet)
	err = filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		list, err := load(path)
		if err != nil {
			return err
		}

		name := strings.ToUpper(filepath.Base(path))
		ref[name] = list
		return nil
	})

	if err != nil {
		log.Fatalf("%v", err)
	}

	geoIPList := new(router.GeoIPList)
	for refName, list := range ref {
		rec := mmdbtype.Map{
			"country": mmdbtype.Map{
				"is_in_european_union": mmdbtype.Bool(false),
				"iso_code":             mmdbtype.String(refName),
				"names": mmdbtype.Map{
					"en": mmdbtype.String(refName),
				},
			},
		}

		list, err = cidrman.MergeIPNets(list)
		if err != nil {
			log.Printf("fail to merge CIDRs: %s", err)
			continue
		}

		cidrList := make([]*router.CIDR, 0)
		for _, ip := range list {
			err = writer.Insert(ip, rec)
			if err != nil {
				log.Fatalf("fail to insert to writer: %v", err)
			}

			var cidr *router.CIDR
			cidr, err = conf.ParseIP(ip.String())
			if err != nil {
				log.Fatalf("fail to convert IP to CIDR: %v", err)
				continue
			}

			cidrList = append(cidrList, cidr)
		}

		geoIPList.Entry = append(geoIPList.Entry, &router.GeoIP{
			CountryCode: refName,
			Cidr:        cidrList,
		})

		listname := strings.ToLower(refName)
		if _, ok := exportMap[listname]; ok {
			exportPlainTextList(listname, list)
		}
	}

	outFh, err := os.Create(filepath.Join(outputDir, "Country.mmdb"))
	if err != nil {
		log.Fatalf("fail to create output file: %v", err)
	}

	_, err = writer.WriteTo(outFh)
	if err != nil {
		log.Fatalf("fail to write to file: %v", err)
	}

	geoIPBytes, err := proto.Marshal(geoIPList)
	if err != nil {
		log.Fatalf("Error marshalling geoip list: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(outputDir, "geoip.dat"), geoIPBytes, 0644); err != nil {
		log.Fatalf("Error writing geoip to file:", err)
	}
}

func exportPlainTextList(refName string, ipList []*net.IPNet) error {
	var data []byte
	for _, ip := range ipList {
		data = append(data, []byte(ip.String()+"\n")...)
	}

	if err := ioutil.WriteFile(filepath.Join(outputDir, refName+".txt"), data, 0644); err != nil {
		return err
	}
	return nil
}
