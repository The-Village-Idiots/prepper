package conf_test

import (
	"testing"
	"time"

	"github.com/ejv2/prepper/conf"
)

const ConfigPath = "./testdata/timetable.json"

var Config conf.Config

func init() {
	cfg, err := conf.NewConfig(ConfigPath)
	if err != nil {
		panic("invalid timetable config: " + err.Error())
	}

	if cfg.TimetableLayout == nil {
		panic("nil timetable layout config")
	}

	Config = cfg
}

func TestWithin(t *testing.T) {
	testdata := []struct {
		Time   string
		Period *conf.Period
		Expect bool
	}{
		// Morning registration [08:35:00 - 09:10:00]
		{"08:35:00", (*Config.TimetableLayout)[0], true},
		{"08:50:00", (*Config.TimetableLayout)[0], true},
		{"09:10:00", (*Config.TimetableLayout)[0], true},
		{"07:00:00", (*Config.TimetableLayout)[0], false},
		{"13:00:00", (*Config.TimetableLayout)[0], false},
	}

	for _, d := range testdata {
		tm, err := time.Parse(time.TimeOnly, d.Time)
		if err != nil {
			panic("bad time value: " + err.Error())
		}

		w := d.Period.Within(tm)
		if w != d.Expect {
			t.Errorf("wrong response: %v within %s (got %v, expect %v)", d.Time, d.Period.Name, w, d.Expect)
		}
	}
}

func TestFindPeriod(t *testing.T) {
	for _, p := range *Config.TimetableLayout {
		t.Logf("%s [%s - %s]", p.Name, time.Time(p.Start).Format(time.TimeOnly), time.Time(p.End).Format(time.TimeOnly))
	}

	testdata := []struct {
		Time         string
		ExpectPeriod string
	}{
		{"11:10:00", "Morning Break"},
		{"08:35:00", "Registration"},
		{"15:55:00", "Period 6"},
		{"19:00:00", "nil"},
	}

	for _, test := range testdata {
		tm, err := time.Parse(time.TimeOnly, test.Time)
		if err != nil {
			panic("bad date in test data: " + err.Error())
		}

		p := Config.TimetableLayout.FindPeriod(tm)
		if p == nil {
			if test.ExpectPeriod != "nil" {
				t.Errorf("%s: nil response for valid period", tm)
			}
			continue
		}

		if p.Name != test.ExpectPeriod {
			t.Errorf("%s: incorrect period returned (expect %v, got %v)", tm, test.ExpectPeriod, p.Name)
		}
	}
}
