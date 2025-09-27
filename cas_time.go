package cas

import (
	"encoding/xml"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
)

// casTime wraps time.Time to provide custom XML marshaling/unmarshaling that
// tolerates the variety of timestamp formats returned by different CAS servers.
type casTime struct {
	time.Time
}

var (
	casTimeLayouts = []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	bracketedZoneExpr = regexp.MustCompile(`\[[^\]]*\]`)
)

func newCASTime(t time.Time) *casTime {
	return &casTime{Time: t}
}

func (ct *casTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}

	if parsed, ok := parseCasTime(raw); ok {
		ct.Time = parsed
	} else {
		if glog.V(1) {
			glog.Warningf("cas: unable to parse authenticationDate %q", raw)
		}
		ct.Time = time.Time{}
	}

	return nil
}

func (ct casTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if ct.Time.IsZero() {
		return nil
	}

	return e.EncodeElement(ct.Time.Format(time.RFC3339), start)
}

func parseCasTime(raw string) (time.Time, bool) {
	cleaned := strings.TrimSpace(bracketedZoneExpr.ReplaceAllString(raw, ""))
	if cleaned == "" {
		return time.Time{}, true
	}

	for _, layout := range casTimeLayouts {
		if ts, err := time.Parse(layout, cleaned); err == nil {
			return ts, true
		}
	}

	return time.Time{}, false
}
