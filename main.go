package main

import (
	"bufio"
	"flag"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	log "github.com/sirupsen/logrus"
)

//"country": mmdbtype.Map{
//"continent": mmdbtype.Map{
//"code": mmdbtype.String("AS"),
//"geoname_id": mmdbtype.Uint32(6255147),
//"names":mmdbtype.Map{"de":mmdbtype.String("Asien"),
//"en":mmdbtype.String("Asia"),
//"es":mmdbtype.String("Asia"),
//"fr":mmdbtype.String("Asie"),
//"ja":mmdbtype.String("アジア"),
//"pt-BR":mmdbtype.String("Ásia"),
//"ru":mmdbtype.String("Азия"),
//"zh-CN":mmdbtype.String("亚洲")},
//},
//"country": mmdbtype.Map{
//"geoname_id":mmdbtype.Uint32(1814991),
//"is_in_european_union":mmdbtype.Bool(false),
//"iso_code":mmdbtype.String("CN"),
//"names":mmdbtype.Map{
//"de":mmdbtype.String("China"),
//"en":mmdbtype.String("China"),
//"es":mmdbtype.String("China"),
//"fr":mmdbtype.String("Chine"),
//"ja":mmdbtype.String("中国"),
//"pt-BR":mmdbtype.String("China"),
//"ru":mmdbtype.String("Китай"),
//"zh-CN":mmdbtype.String("中国"),
//},
//},
//"registered_country": mmdbtype.Map{
//"geoname_id":mmdbtype.Uint32(1814991),
//"is_in_european_union":mmdbtype.Bool(false),
//"iso_code":mmdbtype.String("CN"),
//"names":mmdbtype.Map{
//"de":mmdbtype.String("China"),
//"en":mmdbtype.String("China"),
//"es":mmdbtype.String("China"),
//"fr":mmdbtype.String("Chine"),
//"ja":mmdbtype.String("中国"),
//"pt-BR":mmdbtype.String("China"),
//"ru":mmdbtype.String("Китай"),
//"zh-CN":mmdbtype.String("中国"),
//},
//},
//"traits": mmdbtype.Map{
//"is_anonymous_proxy": mmdbtype.Bool(false),
//"is_satellite_provider":mmdbtype.Bool(false),
//},
//},

var (
	srcFile      string
	reservedFile string
	dstFile      string
	databaseType string
	cnRecord     = mmdbtype.Map{
		"country": mmdbtype.Map{
			"geoname_id":           mmdbtype.Uint32(1814991),
			"is_in_european_union": mmdbtype.Bool(false),
			"iso_code":             mmdbtype.String("CN"),
			"names": mmdbtype.Map{
				"de":    mmdbtype.String("China"),
				"en":    mmdbtype.String("China"),
				"es":    mmdbtype.String("China"),
				"fr":    mmdbtype.String("Chine"),
				"ja":    mmdbtype.String("中国"),
				"pt-BR": mmdbtype.String("China"),
				"ru":    mmdbtype.String("Китай"),
				"zh-CN": mmdbtype.String("中国"),
			},
		},
	}
	reservedRecord = mmdbtype.Map{
		"country": mmdbtype.Map{
			"geoname_id":           mmdbtype.Uint32(9999999),
			"is_in_european_union": mmdbtype.Bool(false),
			"iso_code":             mmdbtype.String("RESERVED"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("Reserved"),
			},
		},
	}
)

func init() {
	flag.StringVar(&srcFile, "s", "ipip_cn.txt", "specify source ip list file")
	flag.StringVar(&reservedFile, "r", "", "specify reserved ip list file")
	flag.StringVar(&dstFile, "d", "Country.mmdb", "specify destination mmdb file")
	flag.StringVar(&databaseType, "t", "GeoIP2-Country", "specify MaxMind database type")
	flag.Parse()
}

func main() {
	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            databaseType,
			RecordSize:              24,
			IncludeReservedNetworks: true,
		},
	)
	if err != nil {
		log.Fatalf("fail to new writer %v\n", err)
	}

	var ipTxtList []string
	var reservedList []string

	fh, err := os.Open(srcFile)
	if err != nil {
		log.Fatalf("fail to open %s\n", err)
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		ipTxtList = append(ipTxtList, scanner.Text())
	}

	if len(reservedFile) != 0 {
		fh, err := os.Open(reservedFile)
		if err != nil {
			log.Fatalf("fail to open %s\n", err)
		}
		defer fh.Close()

		scanner := bufio.NewScanner(fh)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			reservedList = append(reservedList, scanner.Text())
		}
	}

	for _, ip := range parseCIDRs(ipTxtList) {
		err = writer.Insert(ip, cnRecord)
		if err != nil {
			log.Fatalf("fail to insert to writer %v\n", err)
		}
	}

	for _, ip := range parseCIDRs(reservedList) {
		err = writer.Insert(ip, reservedRecord)
		if err != nil {
			log.Fatalf("fail to insert to writer %v\n", err)
		}
	}

	outFh, err := os.Create(dstFile)
	if err != nil {
		log.Fatalf("fail to create output file %v\n", err)
	}

	_, err = writer.WriteTo(outFh)
	if err != nil {
		log.Fatalf("fail to write to file %v\n", err)
	}

}
