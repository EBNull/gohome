package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/renameio/v2"
)

func readLinks(rdr io.Reader) ([]Link, error) {
	links := []Link{}
	err := json.NewDecoder(rdr).Decode(&links)
	return links, err
}

func updateLinksFromRemote(db *LinkDB, path string, cachePath string) error {
	log.Printf("Updating golinks from %s\n", path)
	r, err := http.Get(path)
	if err != nil {
		return fmt.Errorf("Could not download updated golinks: %w", err)
	}
	defer r.Body.Close()
	l, err := readLinks(r.Body)
	if err != nil {
		return fmt.Errorf("Could not parse updated golinks: %w", err)
	}
	stat := db.Update(l)
	log.Printf("Merged %d updated (%d new) golinks\n", len(l), len(stat.Added))
	if len(stat.Added) > 0 {
		added := []string{}
		n := min(5, len(stat.Added))
		for _, l := range stat.Added[:n] {
			added = append(added, l.Display)
		}
		log.Printf("Sample (up to 5) of new links: %s\n", added)
		err = db.WriteCache(cachePath)
		if err != nil {
			log.Printf("Could not write to cache: %s", err)
		}
	}
	return nil
}

type LinkDB struct {
	once  sync.Once
	links map[string]Link
}

type LinkStat struct {
	Added []Link
}

func (db *LinkDB) Len() int {
	return len(db.links)
}

func (db *LinkDB) maybeInit() {
	db.once.Do(func() {
		db.links = map[string]Link{}
	})
}

func (db *LinkDB) Update(links []Link) LinkStat {
	db.maybeInit()
	stat := LinkStat{[]Link{}}
	for _, link := range links {
		maybeFixLinkSource(&link)
		if _, ok := db.links[link.Source]; !ok {
			stat.Added = append(stat.Added, link)
		}
		db.links[link.Source] = link
	}
	return stat
}

func (db *LinkDB) LoadJson(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Printf("Loaded 0 golinks from %s: %s\n", path, err.Error())
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	ls, err := readLinks(f)
	if err != nil {
		return err
	}
	stat := db.Update(ls)
	log.Printf("Loaded %d (%d new) golinks from %s\n", len(ls), len(stat.Added), path)
	return nil
}

func (db *LinkDB) WriteCache(path string) error {
	lns := make([]Link, 0, len(db.links))
	for _, l := range db.links {
		lns = append(lns, l)
	}
	b, err := json.Marshal(lns)
	if err != nil {
		return err
	}
	log.Printf("Writing %d golinks to %s\n", len(db.links), path)
	return renameio.WriteFile(path, b, 0644)
}

func (db *LinkDB) Lookup(name string) *Link {
	if name == "" {
		// Special case the empty string - the db has an entry with one :(
		return nil
	}
	l, ok := db.links[canonicalizeLink(name)]
	if !ok {
		return nil
	}
	return &l
}
