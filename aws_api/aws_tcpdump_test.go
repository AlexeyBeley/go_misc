package aws_api

import (
	"flag"
	"reflect"
	"strings"
	"testing"
)

func TestStartEcho(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
		lg.InfoF("Initializing: %s", config_file_path)
		awsTCPDumpNew, err := AWSTCPDumpNew(config_file_path)
		awsTCPDumpNew.EventsFilter = awsTCPDumpNew.EventsEchoFilter
		awsTCPDumpNew.EventProcessor = awsTCPDumpNew.EventsEchoWriter
		if err != nil {
			panic(err)
		}
		err = awsTCPDumpNew.Start()

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}

func TestStartSubnetFilter(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		config_file_path := "/opt/aws_api_go/AWSTCPDumpConfig.json"
		lg.InfoF("Initializing: %s", config_file_path)
		awsTCPDumpNew, err := AWSTCPDumpNew(config_file_path)
		awsTCPDumpNew.EventsFilter = awsTCPDumpNew.GenerateSubnetFilter([]string{""})
		awsTCPDumpNew.EventProcessor = awsTCPDumpNew.EventsEchoWriterUTCTime
		if err != nil {
			panic(err)
		}
		err = awsTCPDumpNew.Start()

		if err != nil {
			t.Errorf("%v", err)
		}

	})
}

func TestInitConfig(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {

		newAWSTcpDump := &AWSTCPDump{}
		err := newAWSTcpDump.initConfig()
		if err != nil {
			t.Errorf("%v", err)
		}

	})
}

func TestParseArgs(t *testing.T) {
	newAWSTcpDump := &AWSTCPDump{}

	tests := []struct {
		name     string
		args     []string
		expected *AWSTCPDumpConfig
		wantErr  bool
		errStr   string // Expected error substring
	}{
		{
			name:     "Default values",
			args:     []string{},
			expected: &AWSTCPDumpConfig{Region: "", Subnets: []string{}, AWSProfile: "default", LiveRecording: false},
			wantErr:  false,
		}, {
			name:     "subnets",
			args:     []string{"--subnets", "sb-12345"},
			expected: &AWSTCPDumpConfig{Region: "", Subnets: []string{"sb-12345"}, AWSProfile: "default", LiveRecording: false},
			wantErr:  false,
		},{
			name:     "profile",
			args:     []string{"--subnets", "sb-12345", "--profile", "non-default"},
			expected: &AWSTCPDumpConfig{Region: "", Subnets: []string{"sb-12345"}, AWSProfile: "non-default", LiveRecording: false},
			wantErr:  false,
		},{
			name:     "live",
			args:     []string{"--subnets", "sb-12345", "--live", "true"},
			expected: &AWSTCPDumpConfig{Region: "", Subnets: []string{"sb-12345"}, AWSProfile: "default", LiveRecording: true},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new FlagSet for each test to ensure isolation
			fs := flag.NewFlagSet("test-"+tt.name, flag.ContinueOnError) // flag.ContinueOnError prevents os.Exit(2)

			// Suppress flag package output during tests, as it prints to stderr by default
			// This makes test output cleaner.
			fs.SetOutput(new(strings.Builder))

			cfg, err := newAWSTcpDump.ParseArgs(fs, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %t", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.errStr) {
					t.Errorf("ParseFlags() error = %q, want error containing %q", err.Error(), tt.errStr)
				}
				return
			}

			if !reflect.DeepEqual(cfg, tt.expected) {
				t.Errorf("ParseFlags() got = %+v, want %+v", cfg, tt.expected)
			}
		})
	}
}
