package aws_api

import (
	"strconv"
	"testing"
)


func TestGetTaskDefinitions(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := ECSAPINew(nil, nil)

		err := api.GetTaskDefinitions(CallbackEcho, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestGetTaskDefinitionFamilies(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := ECSAPINew(nil, nil)

		err := api.GetTaskDefinitionFamilies(CallbackEcho, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestGetClusters(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		api := ECSAPINew(nil, nil)

		clusters, err := api.GetClusters(nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(clusters) == 0 {
			t.Errorf("clusters found %d", len(clusters))
		}
	})
}

func TestSplitToBulks(t *testing.T) {
	testCases := []struct {
		Name       string
		SourceSize int
		BulkSize   int
		BulksCount int
	}{{Name: "1,10", SourceSize: 1, BulkSize: 10, BulksCount: 1},
		{
			Name: "0,10", SourceSize: 0, BulkSize: 10, BulksCount: 0},
		{Name: "105,10", SourceSize: 105, BulkSize: 10, BulksCount: 11},
		{Name: "100,10", SourceSize: 100, BulkSize: 10, BulksCount: 10},
		{Name: "99,10", SourceSize: 99, BulkSize: 10, BulksCount: 10},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			api := ECSAPINew(nil, nil)

			srcBulk := []*string{}
			for i := range testCase.SourceSize {
				srcBulk = append(srcBulk, StrPtr(strconv.Itoa(i)))
			}

			bulks, err := api.SplitToBulks(srcBulk, testCase.BulkSize)
			if err != nil {
				t.Errorf("%v", err)
			}
			if len(bulks) != testCase.BulksCount {
				t.Errorf("bulks expected %d got %d", testCase.BulksCount, len(bulks))
			}
		})
	}
}
