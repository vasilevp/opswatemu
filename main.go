package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"

	"github.com/alexflint/go-arg"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

const (
	addr = "localhost:11369"

	// openssl ecparam -genkey -name secp384r1 -out server.key
	key = `-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDDz0fiR6VUvH3jBUqxw4ZWZURlDHrobOfUQ/PUu/EZKfTQKhB///JcF
zOWkYq8H372gBwYFK4EEACKhZANiAAQLxSCUYl6P7pvkEslreA+UnGaiwGAfe/Bz
S1LMYDqi60SWzo+pXaUSZmzKxo477bw95YOrIPuPr+qvAC3DXX4z8faCDP+bOtnG
MNmcWwF2ZK1jxtWdnfy6yVGbaK8qQgE=
-----END EC PRIVATE KEY-----
`

	// openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -batch
	cert = `-----BEGIN CERTIFICATE-----
MIICCzCCAZCgAwIBAgIUdZjVwEO2p7yIsQAsvmtMaiwVvfswCgYIKoZIzj0EAwIw
PDELMAkGA1UEBhMCUlUxEzARBgNVBAgMClNvbWUtU3RhdGUxGDAWBgNVBAoMD0Z1
Y2sgT3Bzd2F0IEx0ZDAeFw0yMDA2MDkxMzAzNTZaFw0zMDA2MDcxMzAzNTZaMDwx
CzAJBgNVBAYTAlJVMRMwEQYDVQQIDApTb21lLVN0YXRlMRgwFgYDVQQKDA9GdWNr
IE9wc3dhdCBMdGQwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAQLxSCUYl6P7pvkEslr
eA+UnGaiwGAfe/BzS1LMYDqi60SWzo+pXaUSZmzKxo477bw95YOrIPuPr+qvAC3D
XX4z8faCDP+bOtnGMNmcWwF2ZK1jxtWdnfy6yVGbaK8qQgGjUzBRMB0GA1UdDgQW
BBQqGh7dCgdZ2mVNOeCSkpWNmsNKlDAfBgNVHSMEGDAWgBQqGh7dCgdZ2mVNOeCS
kpWNmsNKlDAPBgNVHRMBAf8EBTADAQH/MAoGCCqGSM49BAMCA2kAMGYCMQCKAze9
K5F/2D00Ge34/XZJ622S7HNEnDvI1OK+SUUrLWwitFSNbnLxZ+awi8J9xZcCMQC7
SCZlcKjAZMyHdK7OGFYbcBUUlvSRPmM63SvxYeOC/Mji8eWtN+QouIufEsXpxzs=
-----END CERTIFICATE-----
`

	statusURL = "https://eapi.opswatgears.com:11369/"
)

// OPSWAT response mock-up

type Info struct {
	HWID string `json:"hwid"`
}

type Reply struct {
	Info        Info   `json:"info"`
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func SuccessResponse(hwid string) Reply {
	return Reply{
		Info: Info{
			HWID: hwid,
		},
		Code:        0,
		Description: "Success",
	}
}

var Config struct {
	HWID string `arg:"-h,env,required" help:"a HWID to emulate; can also be passed in 'HWID' environment variable"`
}

func main() {
	arg.MustParse(&Config)

	r := chi.NewRouter()

	r.Get("/opswat/devinfo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cb := r.URL.Query().Get("callback")

		logrus.WithField("URL", r.URL).Info("Incoming request")

		rd := SuccessResponse(Config.HWID)
		data, err := json.Marshal(rd)
		if err != nil {
			logrus.WithError(err).Error("Cannot marshal response")
		}

		reply := fmt.Sprintf("%s(%s)", cb, data)

		w.Header().Set("Server", "OPSWAT Client")
		w.Header().Set("Content-Type", "text/html")

		if _, err := w.Write([]byte(reply)); err != nil {
			logrus.WithError(err).Error("Cannot write response")
		}

		logrus.WithField("response", reply).Info("Wrote response")
	}))

	r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("It works!\nMake sure to use the same browser to access 'protected' resources")); err != nil {
			logrus.WithError(err).Error("Cannot write response")
		}
	}))

	logrus.WithField("HWID", Config.HWID).Infof("Listening on %s", addr)

	err := runServer(r)
	if err != nil {
		logrus.WithError(err).Error("Cannot run HTTP server; make sure your TLS keys are in the right location")
	}
}

func runServer(r chi.Router) error {
	keyfile, err := ioutil.TempFile("", "fakekey")
	if err != nil {
		return err
	}

	certfile, err := ioutil.TempFile("", "fakecert")
	if err != nil {
		return err
	}

	defer func() {
		if err := os.Remove(keyfile.Name()); err != nil {
			logrus.WithError(err).Errorf("Could not remove temp file %s", keyfile.Name())
		}
		if err := os.Remove(certfile.Name()); err != nil {
			logrus.WithError(err).Errorf("Could not remove temp file %s", keyfile.Name())
		}

		logrus.Info("Cleanup complete")
	}()

	if _, err = keyfile.WriteString(key); err != nil {
		return err
	}
	if _, err = certfile.WriteString(cert); err != nil {
		return err
	}

	if err := keyfile.Close(); err != nil {
		return err
	}
	if err := certfile.Close(); err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	srv := http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		<-quit
		logrus.Info("Shutting down...")
		_ = srv.Shutdown(context.Background())
	}()

	logrus.Infof("Opening status check on %s...", statusURL)
	if err := open.Run(statusURL); err != nil {
		logrus.WithError(err).Errorf("Could not open status check page, please navigate to %s and add the certificate to exceptions", statusURL)
	}

	err = srv.ListenAndServeTLS(certfile.Name(), keyfile.Name())
	if err == http.ErrServerClosed {
		return nil
	}

	return err
}
