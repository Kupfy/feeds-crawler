package crawlstatus

import (
	"database/sql/driver"
	"encoding/json"
)

type CrawlStatus struct {
	value string
}

func (cs CrawlStatus) String() string {
	return cs.value
}

var (
	Created     = CrawlStatus{value: "CREATED"}
	Running     = CrawlStatus{value: "RUNNING"}
	Finished    = CrawlStatus{value: "FINISHED"}
	Errored     = CrawlStatus{value: "ERRORED"}
	PageErrored = CrawlStatus{value: "PAGE_ERRORED"}
)

func NewCrawlStatus(kind string) CrawlStatus {
	return map[string]CrawlStatus{
		Created.value:     Created,
		Running.value:     Running,
		Finished.value:    Finished,
		Errored.value:     Errored,
		PageErrored.value: PageErrored,
	}[kind]
}

func (cs CrawlStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(cs.String())
}

func (cs *CrawlStatus) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*cs = NewCrawlStatus(s)
	return nil
}

func (cs *CrawlStatus) Scan(value interface{}) error {
	if value == nil {
		*cs = CrawlStatus{}
		return nil
	}
	*cs = NewCrawlStatus(value.(string))
	return nil
}

func (cs CrawlStatus) Value() (driver.Value, error) {
	i := NewCrawlStatus(cs.value)
	if i.value == "" {
		return nil, nil
	}
	return i.value, nil
}
