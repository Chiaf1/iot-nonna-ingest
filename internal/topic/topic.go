package topic

import (
	"sync"
	"time"
)

type TopicConfig struct {
	DeviceID      string
	SensorID      string
	TableName     string
	ColumnSchema  map[string]TopicColumnDef
	ValueMapping  map[string]interface{}
	PayloadFormat string
	Qos_mqtt      *uint8
}

type TopicMap struct {
	mu         sync.RWMutex
	items      map[string]TopicConfig
	lastUpdate time.Time
}

type TopicColumnDef struct {
	Column string `json:"column"`
	Type   string `json:"type"`
}

func NewTopicMap() *TopicMap {
	return &TopicMap{
		items: make(map[string]TopicConfig),
	}
}

// Retursn a copy of the topic config if it exist in the topic map. The second return arg is the ok bit of presence in the map,
// if true the key existed in the map if false it didn't
func (t *TopicMap) GetTopicConf(topic string) (TopicConfig, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.items[topic]
	if !ok {
		return TopicConfig{}, false
	}
	return copyTopicConfig(r), ok
}

// Returns the time of the last update of the struct
func (t *TopicMap) GetLastUpdate() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	lu := t.lastUpdate
	return lu
}

// Returns a deep copy of the current topic map
func (t *TopicMap) GetMap() map[string]TopicConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	copy := make(map[string]TopicConfig, len(t.items))
	for k, v := range t.items {
		copy[k] = copyTopicConfig(v)
	}
	return copy
}

// Replace the topic map entierly with the new one
func (t *TopicMap) ReplaceMap(newMap map[string]TopicConfig) {
	newMapCopy := make(map[string]TopicConfig, len(newMap))
	for k, v := range newMap {
		newMapCopy[k] = copyTopicConfig(v)
	}
	t.mu.Lock()
	t.items = newMapCopy
	t.lastUpdate = time.Now()
	t.mu.Unlock()
}

// Returns the list of all topics keys
func (t *TopicMap) GetTopicList() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	list := make([]string, 0, len(t.items))
	for k := range t.items {
		list = append(list, k)
	}
	return list
}

// Helper function to make a deep copy of the topic config item in order to make it thread safe
func copyTopicConfig(tc TopicConfig) TopicConfig {
	cols := make(map[string]TopicColumnDef, len(tc.ColumnSchema))
	for k, v := range tc.ColumnSchema {
		cols[k] = v
	}
	tc.ColumnSchema = cols

	if tc.ValueMapping != nil {
		vm := make(map[string]interface{}, len(tc.ValueMapping))
		for k, v := range tc.ValueMapping {
			vm[k] = v
		}
		tc.ValueMapping = vm
	}
	return tc
}
