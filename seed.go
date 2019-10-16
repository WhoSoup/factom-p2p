package p2p

import (
	"net"

	log "github.com/sirupsen/logrus"
)

type seed struct {
	url string

	logger *log.Entry
}

func newSeed(url string) *seed {
	s := new(seed)
	s.url = url
	s.logger = packageLogger.WithFields(log.Fields{"subpackage": "Seed", "url": url})
	return s
}

func (s *seed) retrieve() []Endpoint {
	eps := make([]Endpoint, 0)

	err := WebScanner(s.url, func(line string) {
		host, port, err := net.SplitHostPort(line)
		if err != nil {
			s.logger.Errorf("Badly formatted line [%s]", line)
			return
		}
		if ep, err := NewEndpoint(host, port); err != nil {
			s.logger.WithError(err).Errorf("Bad peer [%s]", line)
		} else {
			eps = append(eps, ep)
		}
	})

	if err != nil {
		s.logger.WithError(err).Errorf("unable to retrieve data from seed")
	}
	return eps
}
