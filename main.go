package main

import (
	"bufio"
	"flag"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	log "github.com/sirupsen/logrus"

	"github.com/EvilSuperstars/go-cidrman"
)

var (
	dataPath   string
	outputName string
	outputDir  string
	cnRecord   = mmdbtype.Map{
		"country": mmdbtype.Map{
			"is_in_european_union": mmdbtype.Bool(false),
			"iso_code":             mmdbtype.String("CN"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("China"),
			},
		},
	}
	reservedRecord = mmdbtype.Map{
		"country": mmdbtype.Map{
			"is_in_european_union": mmdbtype.Bool(false),
			"iso_code":             mmdbtype.String("RESERVED"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("Reserved"),
			},
		},
	}
)

func init() {
	flag.StringVar(&dataPath, "datapath", "./data", "specify directory which contains ip list files")
	flag.StringVar(&outputName, "outputname", "Country.mmdb", "specify destination mmdb file")
	flag.StringVar(&outputDir, "outputdir", "./", "Directory to place all generated files")
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

		for _, ip := range list {
			err = writer.Insert(ip, rec)
			if err != nil {
				log.Fatalf("fail to insert to writer: %v", err)
			}
		}
	}

	outFh, err := os.Create(filepath.Join(outputDir, outputName))
	if err != nil {
		log.Fatalf("fail to create output file: %v", err)
	}

	_, err = writer.WriteTo(outFh)
	if err != nil {
		log.Fatalf("fail to write to file: %v", err)
	}

}
