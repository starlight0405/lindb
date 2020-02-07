package indexdb

import (
	"github.com/lindb/roaring"

	"github.com/lindb/lindb/constants"
	"github.com/lindb/lindb/pkg/logger"
	"github.com/lindb/lindb/series"
	"github.com/lindb/lindb/tsdb/metadb"
	"github.com/lindb/lindb/tsdb/query"
	"github.com/lindb/lindb/tsdb/tblstore/invertedindex"
)

// InvertedIndex represents the tag's inverted index (tag values => series id list)
type InvertedIndex interface {
	// GetSeriesIDsByTagValueIDs returns series ids by tag value ids for spec metric's tag key
	GetSeriesIDsByTagValueIDs(tagKeyID uint32, tagValueIDs *roaring.Bitmap) (*roaring.Bitmap, error)
	// GetSeriesIDsForTag get series ids for spec metric's tag key
	GetSeriesIDsForTag(tagKeyID uint32) (*roaring.Bitmap, error)
	// GetGroupingContext returns the context of group by
	GetGroupingContext(tagKeyIDs []uint32) (series.GroupingContext, error)
	// buildInvertIndex builds the inverted index for tag value => series ids,
	// the tags is considered as a empty key-value pair while tags is nil.
	buildInvertIndex(namespace, metricName string, tags map[string]string, seriesID uint32)
	// FlushInvertedIndexTo flushes the inverted-index of tag value id=>series ids under tag key
	FlushInvertedIndexTo(flusher invertedindex.Flusher) error
}

type invertedIndex struct {
	store    *TagIndexStore
	metadata metadb.Metadata
}

func newInvertedIndex(metadata metadb.Metadata) InvertedIndex {
	return &invertedIndex{
		metadata: metadata,
		store:    NewTagIndexStore(),
	}
}

// FindSeriesIDsByExpr finds series ids by tag filter expr
func (index *invertedIndex) GetSeriesIDsByTagValueIDs(tagKeyID uint32, tagValueIDs *roaring.Bitmap) (*roaring.Bitmap, error) {
	tagIndex, ok := index.store.Get(tagKeyID)
	if !ok {
		return nil, constants.ErrNotFound
	}
	return tagIndex.getSeriesIDsByTagValueIDs(tagValueIDs), nil
}

// GetSeriesIDsForTag get series ids by tagKeyId
func (index *invertedIndex) GetSeriesIDsForTag(tagKeyID uint32) (*roaring.Bitmap, error) {
	tagIndex, ok := index.store.Get(tagKeyID)
	if !ok {
		return nil, constants.ErrNotFound
	}
	return tagIndex.getAllSeriesIDs(), nil
}

func (index *invertedIndex) GetGroupingContext(tagKeyIDs []uint32) (series.GroupingContext, error) {
	tagKeysLen := len(tagKeyIDs)
	gCtx := query.NewGroupContext(tagKeysLen)
	// validate tagKeys
	for idx, tagKeyID := range tagKeyIDs {
		_, ok := index.store.Get(tagKeyID)
		if !ok {
			return nil, constants.ErrNotFound
		}
		tagValuesEntrySet := query.NewTagValuesEntrySet()
		gCtx.SetTagValuesEntrySet(idx, tagValuesEntrySet)
		//FIXME stone1100
		//tagValuesEntrySet.SetTagValues(tagIndex.getValues())
	}
	return &groupingContext{
		gCtx: gCtx,
	}, nil
}

// buildInvertIndex builds the inverted index for tag value => series ids,
// the tags is considered as a empty key-value pair while tags is nil.
func (index *invertedIndex) buildInvertIndex(namespace, metricName string, tags map[string]string, seriesID uint32) {
	metadataDB := index.metadata.MetadataDatabase()
	tagMetadata := index.metadata.TagMetadata()
	for tagKey, tagValue := range tags {
		tagKeyID, err := metadataDB.GenTagKeyID(namespace, metricName, tagKey)
		if err != nil {
			//FIXME stone1100 add metric???
			indexLogger.Error("gen tag key id fail, ignore index build for this tag key",
				logger.String("key", tagKey), logger.Error(err))
			continue
		}
		tagIndex, ok := index.store.Get(tagKeyID)
		if !ok {
			tagIndex = newTagIndex()
			index.store.Put(tagKeyID, tagIndex)
		}
		tagValueID, err := tagMetadata.GenTagValueID(tagKeyID, tagValue)
		if err != nil {
			//FIXME stone1100 add metric???
			indexLogger.Error("gen tag value id fail, ignore index build for this tag key",
				logger.String("key", tagKey), logger.String("value", tagValue), logger.Error(err))
			continue
		}
		tagIndex.buildInvertedIndex(tagValueID, seriesID)
	}
}

// FlushInvertedIndexTo flushes the inverted-index of tag value id=>series ids under tag key
func (index *invertedIndex) FlushInvertedIndexTo(flusher invertedindex.Flusher) error {
	//seriesIDBitmap := index.store.tagKeyIDs
	//for idx, highKey := range seriesIDBitmap.GetHighKeys() {
	//	container := seriesIDBitmap.GetContainer(highKey)
	//	tagIndexes := index.store.indexes[idx]
	//	it := container.PeekableIterator()
	//	i := 0
	//	for it.HasNext() {
	//		lowKeyID := it.Next()
	//		tagIndex := tagIndexes[i]
	//		tagValues := tagIndex.getValues()
	//		for tagValue, seriesIDs := range tagValues {
	//			flusher.FlushTagValue(tagValue, seriesIDs)
	//		}
	//		tagKeyID := uint32(lowKeyID) | uint32(highKey)
	//		if err := flusher.FlushTagKeyID(tagKeyID); err != nil {
	//			return err
	//		}
	//	}
	//}
	//return nil
	//FIXME stone1100
	panic("need to impl")
}