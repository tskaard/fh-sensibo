package sensibo

import (
	"net/url"
	"testing"
)

func TestResourceUrl(t *testing.T) {
	s := NewSensibo("sample-key")
	v := s.resourceUrl("users/me/pods", url.Values{
		"fields": []string{"id,room"},
	})
	if v != "https://home.sensibo.com/api/v2/users/me/pods?apiKey=sample-key&fields=id%2Croom" {
		t.Fatalf("bad: %v", v)
	}
}
