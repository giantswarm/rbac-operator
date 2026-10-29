package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/rbac-operator/flag"
)

func Test_Service(t *testing.T) {
	testCases := []struct {
		Name                          string
		WriteAllCustomerGroups        []map[string]string
		ReadAllCustomerGroups         []map[string]string
		WriteAllGiantswarmGroups      []map[string]string
		LegacyWriteAllCustomerGroup   string
		LegacyWriteAllGiantswarmGroup string
		// RawAccessGroups, when set, is used as the raw accessGroups value instead of the group lists.
		RawAccessGroups any
		ExpectedError   error
	}{
		{
			Name:                     "case 0: Instantiate Service with defined access group lists",
			WriteAllCustomerGroups:   []map[string]string{{"name": "customer:acme:Employees"}},
			ReadAllCustomerGroups:    []map[string]string{{"name": "customer:acme:Observers"}},
			WriteAllGiantswarmGroups: []map[string]string{{"name": "giantswarm:giantswarm:giantswarm-admins"}},
		},
		{
			Name:                          "case 1: Instantiate Service with legacy access groups",
			LegacyWriteAllCustomerGroup:   "customer:acme:Employees",
			LegacyWriteAllGiantswarmGroup: "giantswarm:giantswarm:giantswarm-admins",
		},
		{
			Name:          "case 2: Fail to instantiate service without any access groups defined",
			ExpectedError: invalidConfigError,
		},
		{
			Name:                          "case 3: Fail to instantiate service with malformed access groups",
			RawAccessGroups:               map[string]any{"writeAllCustomerGroups": "customer:acme:Employees"},
			LegacyWriteAllGiantswarmGroup: "giantswarm:giantswarm:giantswarm-admins",
			ExpectedError:                 invalidConfigError,
		},
		{
			Name: "case 4: Fail to instantiate service with unknown access group keys",
			RawAccessGroups: map[string]any{
				"writeAllCustomerGroup":    []map[string]string{{"name": "customer:acme:Employees"}},
				"writeAllGiantswarmGroups": []map[string]string{{"name": "giantswarm:giantswarm:giantswarm-admins"}},
			},
			ExpectedError: invalidConfigError,
		},
		{
			Name: "case 5: Fail to instantiate service with unknown access group fields",
			RawAccessGroups: map[string]any{
				"writeAllCustomerGroups":   []map[string]string{{"nmae": "customer:acme:Employees"}},
				"writeAllGiantswarmGroups": []map[string]string{{"name": "giantswarm:giantswarm:giantswarm-admins"}},
			},
			ExpectedError: invalidConfigError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			server := mockKubernetesApiServer()
			defer server.Close()

			loggerConfig := micrologger.Config{}
			logger, err := micrologger.New(loggerConfig)

			if err != nil {
				t.Fatalf("Failed to init logger: %s", err)
			}

			v := viper.New()
			v.Set("service.kubernetes.address", server.URL)

			if tc.RawAccessGroups != nil {
				v.Set("service.accessGroups", tc.RawAccessGroups)
			}
			if len(tc.WriteAllCustomerGroups) > 0 {
				v.Set("service.accessGroups.writeAllCustomerGroups", tc.WriteAllCustomerGroups)
			}
			if len(tc.WriteAllGiantswarmGroups) > 0 {
				v.Set("service.accessGroups.writeAllGiantswarmGroups", tc.WriteAllGiantswarmGroups)
			}
			if len(tc.ReadAllCustomerGroups) > 0 {
				v.Set("service.accessGroups.readAllCustomerGroups", tc.ReadAllCustomerGroups)
			}
			if tc.LegacyWriteAllCustomerGroup != "" {
				v.Set("service.writeAllCustomerGroup", tc.LegacyWriteAllCustomerGroup)
			}
			if tc.LegacyWriteAllGiantswarmGroup != "" {
				v.Set("service.writeAllGiantswarmGroup", tc.LegacyWriteAllGiantswarmGroup)
			}

			config := Config{
				Flag:   flag.New(),
				Logger: logger,
				Viper:  v,
			}

			_, err = New(config)

			if tc.ExpectedError != nil && err == nil {
				t.Fatalf("Failed to receive the expected error: %s", tc.ExpectedError)
			}
			if tc.ExpectedError != err && microerror.Cause(err) != tc.ExpectedError {
				t.Fatalf("Received unexpected error: %s", err)
			}
		})
	}
}

func mockKubernetesApiServer() *httptest.Server {
	hf := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}
	return httptest.NewServer(http.HandlerFunc(hf))
}
