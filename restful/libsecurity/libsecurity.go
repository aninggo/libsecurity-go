package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	app "github.com/ibm-security-innovation/libsecurity-go/app/token"
	en "github.com/ibm-security-innovation/libsecurity-go/entity"
	logger "github.com/ibm-security-innovation/libsecurity-go/logger"
	"github.com/ibm-security-innovation/libsecurity-go/restful/accounts-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/acl-restful"
	cr "github.com/ibm-security-innovation/libsecurity-go/restful/common-restful"
	enRestful "github.com/ibm-security-innovation/libsecurity-go/restful/entity-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/libsecurity-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/ocra-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/otp-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/password-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/storage-restful"
	ss "github.com/ibm-security-innovation/libsecurity-go/storage"
)

const (
	amToken            = "accountManager"
	umToken            = "um"
	aclToken           = "acl"
	appAclToken        = "appAcl"
	otpToken           = "otp"
	ocraToken          = "ocra"
	passwordToken      = "password"
	secureStorageToken = "secureStorage"

	fullToken  = "full"
	basicToken = "basic"
	noneToken  = "none"

	httpsStr = "https"
)

var (
	configOptions []string

	verifyKey                                   *rsa.PublicKey
	signKey                                     *rsa.PrivateKey
	loginKey                                    []byte
	host, protocol, sslServerCert, sslServerKey *string
	generateJSONFlag                            *bool
)

type config map[string]string

func usage() {
	_, file := filepath.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "usage: %v.go\n", file)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nConfiguration file tokens are: %v\n", configOptions)
	fmt.Fprintf(os.Stderr, "Options to configure: ('%v', '%v')\n", basicToken, fullToken)
	fmt.Fprintf(os.Stderr, "Note: The option '%v' is relevant only for %v\n", fullToken, amToken)
	os.Exit(2)
}

func init() {
	cr.ServicePathPrefix = "/forewind/app"
	configOptions = []string{amToken, umToken, aclToken, appAclToken, otpToken, ocraToken, passwordToken, secureStorageToken}
	protocol = flag.String("protocol", "https", "Using protocol: http ot https")
	host = flag.String("host", "127.0.0.1:5443", "Listening host")
	generateJSONFlag = flag.Bool("generate", false, "generate static json")
	sslServerCert = flag.String("server-cert", "./dist/server.crt", "SSL server certificate file path for https")
	sslServerKey = flag.String("server-key", "./dist/server.key", "SSL server key file path for https")
}

func runRestAPI(wsContainer *restful.Container) {
	config := swagger.Config{
		WebServices:     wsContainer.RegisteredWebServices(),
		WebServicesUrl:  "/", // host + port,
		ApiPath:         "/forewind/security.json",
		SwaggerPath:     "/forewind/doc/",
		SwaggerFilePath: "./dist",
		// TODO set it Title:           "libsecurity-go",
		// TODO set it Description:     "The libsecurity-go tool is for",
	}

	swagger.RegisterSwaggerService(config, wsContainer)
	if *generateJSONFlag {
		go generateJSON(config.ApiPath, config.SwaggerFilePath+"/")
	}
	log.Printf("start listening on %v", *host)
	var err error
	if strings.HasPrefix(strings.ToLower(*protocol), httpsStr) {
		err = http.ListenAndServeTLS(*host, *sslServerCert, *sslServerKey, wsContainer)
	} else {
		err = http.ListenAndServe(*host, wsContainer)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func readConfigFile(configFile string) (config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var configData config
	err = json.Unmarshal(data, &configData)
	if err != nil {
		return nil, err
	}
	logger.Trace.Printf("The config data: %v", configData)
	return configData, nil
}

func registerComponents(configFile string, secureKeyFilePath string, privateKeyFilePath string, usersDataPath string) {
	conf, err := readConfigFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error while reading configuration file '%v', error: %v\n", configFile, err)
		os.Exit(1)
	}
	wsContainer := restful.NewContainer()
	usersList := en.New()

	//	amUsers := am.NewAmUsersList()
	signKey, verifyKey = app.SetupAToken(privateKeyFilePath)
	loginKey = ss.GetSecureKey(secureKeyFilePath)

	st := libsecurityRestful.NewLibsecurityRestful()
	st.SetData(usersList, loginKey, verifyKey, signKey, nil)

	l := accountsRestful.NewAmRestful()
	l.SetData(st)
	if conf[amToken] == fullToken {
		l.RegisterFull(wsContainer)
	} else { // login is mandatory
		l.RegisterBasic(wsContainer)
	}

	um := enRestful.NewEnRestful()
	um.SetData(st)
	if conf[umToken] != noneToken {
		um.RegisterBasic(wsContainer)
	}

	a := aclRestful.NewAclRestful()
	a.SetData(st)
	if conf[aclToken] == basicToken || conf[appAclToken] == basicToken {
		a.RegisterBasic(wsContainer)
	}

	p := otpRestful.NewOtpRestful()
	p.SetData(st)
	if conf[otpToken] == basicToken {
		p.RegisterBasic(wsContainer)
	}

	o := ocraRestful.NewOcraRestful()
	o.SetData(st)
	if conf[ocraToken] == basicToken {
		o.RegisterBasic(wsContainer)
	}

	pwd := passwordRestful.NewPwdRestful()
	pwd.SetData(st)
	if conf[passwordToken] == basicToken {
		pwd.RegisterBasic(wsContainer)
	}

	ss := storageRestful.NewSsRestful()
	ss.SetData(st)
	if conf[secureStorageToken] == basicToken {
		ss.RegisterBasic(wsContainer)
	}

	st.RegisterBasic(wsContainer)

	err = en.LoadInfo(usersDataPath, loginKey, usersList)
	if err != nil {
		fmt.Println("Load info error:", err)
	}
	runRestAPI(wsContainer)
}

func generateJSON(path string, distPath string) {
	var obj map[string]interface{}
	baseURL := fmt.Sprintf("%v://%v%v", *protocol, *host, path)
	fileFmt := "%v/%v"

	time.Sleep(100 * time.Millisecond)
	_, jsonD, _ := cr.HTTPDataMethod(cr.HTTPGetStr, baseURL, "")
	err := json.Unmarshal([]byte(jsonD), &obj)
	if err != nil {
		log.Fatal(err)
	}
	var prefix, p1, file string
	for i, v := range obj["apis"].([]interface{}) {
		casted := v.(map[string]interface{})
		url1 := fmt.Sprintf("%v%v", baseURL, casted["path"])
		_, u, _ := cr.HTTPDataMethod(cr.HTTPGetStr, url1, "")
		p1, file = filepath.Split(fmt.Sprintf("%v", casted["path"]))
		if i == 0 {
			prefix = strings.Replace(p1, "/", distPath, 1)
			err := os.MkdirAll(prefix, 0777)
			if err != nil {
				log.Fatalf("Fatal error while generating static JSON path: %v", err)
			}
		}
		ioutil.WriteFile(fmt.Sprintf(fileFmt, prefix, file), []byte(u), 0777)
	}
	_, file = filepath.Split(path)
	prefix1 := strings.Replace(p1, "/", "/../", 1)
	obj["apiVersion"] = "2.02"
	a := obj["info"].(map[string]interface{})
	a["title"] = "Libsecurity API"
	j, _ := json.Marshal(obj)
	newS := strings.Replace(string(j), p1, prefix1, -1)
	ioutil.WriteFile(fmt.Sprintf(fileFmt, distPath, file), []byte(newS), 0777)
}

func main() {
	privateKeyFilePath := flag.String("rsa-private", "./dist/key.private", "RSA private key file path")
	secureKeyFilePath := flag.String("secure-key", "./dist/secureKey", "password to encrypt the secure storage")
	usersDataPath := flag.String("storage-file", "./dist/data.txt", "persistence storage file")
	configFile := flag.String("config-file", "./config.json", "Configuration information file")
	flag.Parse()
	if flag.NArg() > 0 {
		usage()
	}
	registerComponents(*configFile, *secureKeyFilePath, *privateKeyFilePath, *usersDataPath)
}
