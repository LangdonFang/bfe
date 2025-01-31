// Copyright (c) 2019 Baidu, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bfe_conf

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
)

import (
	"github.com/baidu/go-lib/log"
)

import (
	"github.com/baidu/bfe/bfe_tls"
	"github.com/baidu/bfe/bfe_util"
)

var TlsVersionMap = map[string]uint16{
	"VersionSSL30": bfe_tls.VersionSSL30,
	"VersionTLS10": bfe_tls.VersionTLS10,
	"VersionTLS11": bfe_tls.VersionTLS11,
	"VersionTLS12": bfe_tls.VersionTLS12,
}

var CurvesMap = map[string]bfe_tls.CurveID{
	"CurveP256": bfe_tls.CurveP256,
	"CurveP384": bfe_tls.CurveP384,
	"CurveP521": bfe_tls.CurveP521,
}

var CipherSuitesMap = map[string]uint16{
	"TLS_RSA_WITH_RC4_128_SHA":                          bfe_tls.TLS_RSA_WITH_RC4_128_SHA,
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                     bfe_tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA":                      bfe_tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA":                      bfe_tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":                  bfe_tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":              bfe_tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":              bfe_tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                    bfe_tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":               bfe_tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":                bfe_tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":                bfe_tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":             bfe_tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":           bfe_tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":       bfe_tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256":     bfe_tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

const (
	EquivCipherSep = "|" // separator for equivalent ciphers string
)

type ConfigHttpsBasic struct {
	ServerCertConf string // config for server cert and key
	TlsRuleConf    string // config for server tls rule

	CipherSuites     []string // supported cipher suites
	CurvePreferences []string // curve perference

	MaxTlsVersion string // max tls version supported
	MinTlsVersion string // min tls version supported

	EnableSslv2ClientHello bool // support sslv2 client hello for backward compatibility

	ClientCABaseDir string // client root CAs base directory
}

func (cfg *ConfigHttpsBasic) Check(confRoot string) error {
	// check cert file conf
	err := certConfCheck(cfg, confRoot)
	if err != nil {
		return err
	}

	// check cert rule conf
	err = certRuleCheck(cfg, confRoot)
	if err != nil {
		return err
	}

	// check CipherSuites
	for _, cipherGroup := range cfg.CipherSuites {
		ciphers := strings.Split(cipherGroup, EquivCipherSep)
		for _, cipher := range ciphers {
			if _, ok := CipherSuitesMap[cipher]; !ok {
				return fmt.Errorf("cipher (%s) not support", cipher)
			}
		}
	}

	// check CurvePreferences
	for _, curve := range cfg.CurvePreferences {
		if _, ok := CurvesMap[curve]; !ok {
			return fmt.Errorf("curve (%s) not support", curve)
		}
	}

	// check tls version
	err = tlsVersionCheck(cfg)
	if err != nil {
		return err
	}

	// check client CA certificate base dir
	if len(cfg.ClientCABaseDir) == 0 {
		return fmt.Errorf("ClientCABaseDir empty")
	}

	return nil
}

func certConfCheck(cfg *ConfigHttpsBasic, confRoot string) error {
	if cfg.ServerCertConf == "" {
		log.Logger.Warn("ServerCertConf not set, use default value")
		cfg.ServerCertConf = "tls_conf/server_cert_conf.data"
	}
	cfg.ServerCertConf = bfe_util.ConfPathProc(cfg.ServerCertConf, confRoot)
	return nil
}

func certRuleCheck(cfg *ConfigHttpsBasic, confRoot string) error {
	if cfg.TlsRuleConf == "" {
		log.Logger.Warn("TlsRuleConf not set, use default value")
		cfg.TlsRuleConf = "tls_conf/tls_rule_conf.data"
	}
	cfg.TlsRuleConf = bfe_util.ConfPathProc(cfg.TlsRuleConf, confRoot)
	return nil
}

func tlsVersionCheck(cfg *ConfigHttpsBasic) error {
	if len(cfg.MaxTlsVersion) == 0 {
		cfg.MaxTlsVersion = "VersionTLS12"
	}
	if len(cfg.MinTlsVersion) == 0 {
		cfg.MinTlsVersion = "VersionSSL30"
	}

	maxTlsVer, ok := TlsVersionMap[cfg.MaxTlsVersion]
	if !ok {
		return fmt.Errorf("Max TLS version(%s) not support", cfg.MaxTlsVersion)
	}
	minTlsVer, ok := TlsVersionMap[cfg.MinTlsVersion]
	if !ok {
		return fmt.Errorf("Min TLS version(%s) not support", cfg.MinTlsVersion)
	}

	if maxTlsVer < minTlsVer {
		return fmt.Errorf("Max TLS version should not less than Min TLS version")
	}

	return nil
}

// LoadClientCAFile loades client ca certificate in PEM format
func LoadClientCAFile(path string) (*x509.CertPool, error) {
	roots := x509.NewCertPool()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots.AppendCertsFromPEM(data)
	return roots, nil
}

func GetCurvePreferences(curveConf []string) ([]bfe_tls.CurveID, error) {
	curvePreferences := make([]bfe_tls.CurveID, 0, len(curveConf))
	for _, curveStr := range curveConf {
		curve, ok := CurvesMap[curveStr]
		if !ok {
			return nil, fmt.Errorf("ellptic curve (%s) not support", curveStr)
		}
		curvePreferences = append(curvePreferences, curve)
	}
	return curvePreferences, nil
}

func GetCipherSuites(cipherConf []string) ([]uint16, []uint16, error) {
	cipherSuites := make([]uint16, 0, len(cipherConf))
	cipherSuitesPriority := make([]uint16, 0, len(cipherConf))

	for i, cipherGroup := range cipherConf {
		ciphers := strings.Split(cipherGroup, EquivCipherSep)
		for _, cipher := range ciphers {
			cipherSuite, ok := CipherSuitesMap[cipher]
			if !ok {
				return nil, nil, fmt.Errorf("ciphersuite (%s) not support", cipher)
			}
			cipherSuites = append(cipherSuites, cipherSuite)
			cipherSuitesPriority = append(cipherSuitesPriority, uint16(i))
		}
	}

	return cipherSuites, cipherSuitesPriority, nil
}

func GetTlsVersion(cfg *ConfigHttpsBasic) (maxVer, minVer uint16) {
	maxTlsVersion, ok := TlsVersionMap[cfg.MaxTlsVersion]
	if !ok {
		maxTlsVersion = bfe_tls.VersionTLS12
	}

	minTlsVersion, ok := TlsVersionMap[cfg.MinTlsVersion]
	if !ok {
		minTlsVersion = bfe_tls.VersionSSL30
	}

	return maxTlsVersion, minTlsVersion
}
