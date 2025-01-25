package qdranttypes

type CollectionName string
type VectorSize int

const (
	COLLECTION_NAME_BASIC_MESSAGES      CollectionName = "basic_messages"
	COLLECTION_NAME_REFLECTION_MESSAGES CollectionName = "reflection_messages"
)

const (
	VECTOR_SIZE_BASIC_MESSAGE VectorSize = 1536
)
