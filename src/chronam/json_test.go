package chronam

import (
	"testing"
)

func TestBatchJSON(t *testing.T) {
	var encoded = `
{
	"name": "batch_oru_quartz_ver01",
	"url": "http://oregonnews.uoregon.edu/batches/batch_oru_quartz_ver01.json",
	"page_count": 10705,
	"awardee": {
		"url": "http://oregonnews.uoregon.edu/awardees/oru.json",
		"name": "University of Oregon Libraries; Eugene, OR"
	},
	"lccns": [
		"sn94052320", "sn2002060538", "sn94052319", "2015260100", "sn93051660",
		"sn96088356", "sn84022643", "sn97071110", "sn84022650", "sn97071028"
	],
	"ingested": "2016-05-13T10:28:21-07:00",
	"issues": [
		{
			"url": "http://oregonnews.uoregon.edu/lccn/sn84022643/1868-10-03/ed-1.json",
			"date_issued": "1868-10-03",
			"title": {
				"url": "http://oregonnews.uoregon.edu/lccn/sn84022643.json", "name": "The Albany register."
			}
		},
		{
			"url": "http://oregonnews.uoregon.edu/lccn/sn84022643/1868-11-28/ed-1.json",
			"date_issued": "1868-11-28",
			"title": {
				"url": "http://oregonnews.uoregon.edu/lccn/sn84022643.json", "name": "The Albany register."
			}
		},
		{
			"url": "http://oregonnews.uoregon.edu/lccn/sn84022643/1868-12-05/ed-1.json",
			"date_issued": "1868-12-05",
			"title": {
				"url": "http://oregonnews.uoregon.edu/lccn/sn84022643.json", "name": "The Albany register."
			}
		},
		{
			"url": "http://oregonnews.uoregon.edu/lccn/sn84022643/1868-12-12/ed-1.json",
			"date_issued": "1868-12-12",
			"title": {
				"url": "http://oregonnews.uoregon.edu/lccn/sn84022643.json", "name": "The Albany register."
			}
		}
	]
}
		`

	var b, err = parseBatchJSON([]byte(encoded))
	if err != nil {
		t.Fatalf("JSON wouldn't parse: %s", err)
	}

	var actualName = b.Name
	var expectedName = "batch_oru_quartz_ver01"
	if actualName != expectedName {
		t.Fatalf("Batch name was %#v; expected to see %#v", actualName, expectedName)
	}

	if len(b.Issues) != 4 {
		t.Fatalf("Expected to find 4 issues, but got %d", len(b.Issues))
	}

	var actualDate = b.Issues[2].Date
	var expectedDate = "1868-12-05"
	if actualDate != expectedDate {
		t.Fatalf("Third issue date was %#v; expected to see %#v", actualDate, expectedDate)
	}
}
