package workloads


import (
	"sync"
	"math/rand"
	"strconv"
	"crypto/md5"
	"encoding/hex"
	"bytes"

	"databases"
)


type Config struct {
	CreatePercentage int	// shorthand "c"
	ReadPercentage int		// shorthand "r"
	UpdatePercentage int	// shorthand "u"
	DeletePercentage int	// shorthand "d"
	QueryPercentage int		// shorthand "q"
	Records int64
	Operations int64
	ValueSize int
	IndexableFields int
	Workers int
}


type State struct {
	Operations, Records int64
	Errors []string
}


func hash(in_string string) string {
	h := md5.New()
	h.Write([]byte(in_string))
	return hex.EncodeToString(h.Sum(nil))
}


func GenerateNewKey(current_records int64) string {
	str_current_records := strconv.FormatInt(current_records, 10)
	return hash(str_current_records)
}


func GenerateExistingKey(current_records, current_operations int64) string {
	rand.Seed(current_operations)
	rand_record := rand.Int63n(current_records)
	str_rand_record := strconv.FormatInt(rand_record, 10)
	return hash(str_rand_record)
}


func GenerateValue(key string, indexable_fields, size int) map[string]interface{} {
	if indexable_fields >= 20 {
		panic("Too much fields! It must be less than 20")
	}
	map_value := make(map[string]interface{})
	for i := 0; i < indexable_fields; i++ {
		fieldName := "field" + strconv.Itoa(i)
		map_value[fieldName] = fieldName + "-" +key[i:i + 10]
	}
	fieldName := "field" + strconv.Itoa(indexable_fields)
	var buffer bytes.Buffer
	var body_hash string = hash(key)
	iterations := (size - len(fieldName + "-" + key[:10]) * indexable_fields) / 32
	for i := 0; i < iterations; i++ {
		buffer.WriteString(body_hash)
	}
	map_value[fieldName] = buffer.String()
	return map_value
}



func PrepareBatch(config Config) []string {
	operations := make([]string, 0, 100)
	rand_operations := make([]string, 100, 100)
	for i := 0; i < config.CreatePercentage; i++ {
		operations = append(operations, "c")
	}
	for i := 0; i < config.ReadPercentage; i++ {
		operations = append(operations, "r")
	}
	for i := 0; i < config.UpdatePercentage; i++ {
		operations = append(operations, "u")
	}
	for i := 0; i < config.DeletePercentage; i++ {
		operations = append(operations, "d")
	}
	for i := 0; i < config.QueryPercentage; i++ {
		operations = append(operations, "q")
	}
	if len(operations) != 100 {
		panic("Wrong workload configuration: sum of percentages is not equal 100")
	}
	for i, rand_i := range rand.Perm(100) {
		rand_operations[i] = operations[rand_i]
	}
	return rand_operations
}


func DoBatch(db databases.Database, config Config, state *State) {
	var key string
	var value map[string]interface{}
	var status error
	var batch = PrepareBatch(config)

	for _, v := range batch {
		switch v {
		case "c":
			state.Records ++
			key = GenerateNewKey(state.Records)
			value = GenerateValue(key, config.IndexableFields, config.ValueSize)
			status = db.Create(key, value)
		case "r":
			key = GenerateExistingKey(state.Records, state.Operations)
			status = db.Read(key)
		case "u":
			key = GenerateExistingKey(state.Records, state.Operations)
			value = GenerateValue(key, config.IndexableFields, config.ValueSize)
			status = db.Update(key, value)
		case "d":
			key = GenerateExistingKey(state.Records, state.Operations)
			status = db.Delete(key)
		}
		if status != nil {
			state.Errors = append(state.Errors, v)
		}
	}
}

func RunWorkload(database databases.Database, config Config, state *State, wg *sync.WaitGroup) {
	for state.Operations < config.Operations {
		state.Operations += 100
		DoBatch(database, config, state)
	}
	wg.Done()
}
