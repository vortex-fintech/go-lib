package schemaregistry

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/twmb/franz-go/pkg/sr"
	"google.golang.org/protobuf/proto"
)

var (
	ErrSubjectRequired = errors.New("subject is required")
	ErrNilMessage      = errors.New("protobuf message is nil")
	ErrSchemaRequired  = errors.New("protobuf schema text is required for first serialize")
	ErrSchemaNotCached = errors.New("schema id is not cached for subject; call SerializeWithSchema first")

	confluentHeader = new(sr.ConfluentHeader)
)

type ProtoSerializer struct {
	registry RegistryClient
	cache    sync.Map
}

type subjectSchemaCache struct {
	id      int
	schema  string
	refsKey string
}

func NewProtoSerializer(registry RegistryClient) *ProtoSerializer {
	return &ProtoSerializer{registry: registry}
}

// Serialize serializes protobuf payload using cached schema ID for subject.
// For first write, call SerializeWithSchema / SerializeWithSchemaRefs.
func (s *ProtoSerializer) Serialize(subject string, message proto.Message) ([]byte, int, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, 0, ErrSubjectRequired
	}
	if message == nil {
		return nil, 0, ErrNilMessage
	}

	data, err := proto.Marshal(message)
	if err != nil {
		return nil, 0, err
	}

	if cachedRaw, ok := s.cache.Load(subject); ok {
		schemaID := cachedRaw.(subjectSchemaCache).id
		return createWireFormat(data, schemaID, []int{0}), schemaID, nil
	}

	return nil, 0, ErrSchemaNotCached
}

// SerializeWithSchema registers schema (if needed), caches ID and serializes payload.
func (s *ProtoSerializer) SerializeWithSchema(subject, schema string, message proto.Message) ([]byte, int, error) {
	return s.SerializeWithSchemaRefs(subject, schema, nil, message)
}

// SerializeWithSchemaRefs registers schema with references (if needed), caches ID and serializes payload.
func (s *ProtoSerializer) SerializeWithSchemaRefs(subject, schema string, refs []SchemaReference, message proto.Message) ([]byte, int, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, 0, ErrSubjectRequired
	}
	if message == nil {
		return nil, 0, ErrNilMessage
	}

	data, err := proto.Marshal(message)
	if err != nil {
		return nil, 0, err
	}

	refsKey := referencesCacheKey(refs)
	if cachedRaw, ok := s.cache.Load(subject); ok {
		cached := cachedRaw.(subjectSchemaCache)
		if cached.schema == schema && cached.refsKey == refsKey {
			return createWireFormat(data, cached.id, []int{0}), cached.id, nil
		}
	}

	if strings.TrimSpace(schema) == "" {
		return nil, 0, ErrSchemaRequired
	}

	schemaID, err := s.registry.RegisterSchemaWithRefs(subject, schema, refs)
	if err != nil {
		return nil, 0, err
	}

	s.cache.Store(subject, subjectSchemaCache{id: schemaID, schema: schema, refsKey: refsKey})
	return createWireFormat(data, schemaID, []int{0}), schemaID, nil
}

func referencesCacheKey(refs []SchemaReference) string {
	if len(refs) == 0 {
		return ""
	}

	b := strings.Builder{}
	for _, r := range refs {
		b.WriteString(r.Name)
		b.WriteByte('|')
		b.WriteString(r.Subject)
		b.WriteByte('|')
		b.WriteString(strconv.Itoa(r.Version))
		b.WriteByte(';')
	}
	return b.String()
}

func createWireFormat(data []byte, schemaID int, messageIndex []int) []byte {
	buf, _ := confluentHeader.AppendEncode(nil, schemaID, messageIndex)
	return append(buf, data...)
}
