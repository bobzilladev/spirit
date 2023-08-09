package table

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompositeChunkerCompositeBinary(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS composite_binary_t1")
	runSQL(t, `CREATE TABLE composite_binary_t1 (
		a varbinary(40) NOT NULL,
		b varbinary(40) NOT NULL,
		c int NOT NULL,
		PRIMARY KEY (a,b)
	)`)
	runSQL(t, `INSERT INTO composite_binary_t1 (a, b, c) SELECT UUID(), UUID(), 1 FROM dual`)                                                                                      //nolint: dupword
	runSQL(t, `INSERT INTO composite_binary_t1 (a, b, c) SELECT UUID(), UUID(), 1 FROM composite_binary_t1 a JOIN composite_binary_t1 b JOIN composite_binary_t1 c LIMIT 1000000`) //nolint: dupword
	runSQL(t, `INSERT INTO composite_binary_t1 (a, b, c) SELECT UUID(), UUID(), 1 FROM composite_binary_t1 a JOIN composite_binary_t1 b JOIN composite_binary_t1 c LIMIT 1000000`) //nolint: dupword
	runSQL(t, `INSERT INTO composite_binary_t1 (a, b, c) SELECT UUID(), UUID(), 1 FROM composite_binary_t1 a JOIN composite_binary_t1 b JOIN composite_binary_t1 c LIMIT 1000000`) //nolint: dupword
	runSQL(t, `INSERT INTO composite_binary_t1 (a, b, c) SELECT UUID(), UUID(), 1 FROM composite_binary_t1 a JOIN composite_binary_t1 b JOIN composite_binary_t1 c LIMIT 1000000`) //nolint: dupword

	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "composite_binary_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))

	// Assert that the types are correct.
	assert.Equal(t, []string{"varbinary", "varbinary"}, t1.keyColumnsMySQLTp)
	assert.Equal(t, binaryType, t1.keyDatums[0])
	assert.Equal(t, binaryType, t1.keyDatums[1])

	chunker, err := NewChunker(t1, ChunkerDefaultTarget, logrus.New())
	assert.NoError(t, err)
	assert.IsType(t, &chunkerComposite{}, chunker)

	assert.NoError(t, chunker.Open())

	chunk, err := chunker.Next()
	assert.NoError(t, err)
	assert.NotContains(t, "`a` >= ", chunk.String()) // first chunk is special
	upperBound := chunk.UpperBound.Value

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound := upperBound
	upperBound = chunk.UpperBound.Value
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("((`a` > %s)\n OR (`a` = %s AND `b` >= %s)) AND ((`a` < %s)\n OR (`a` = %s AND `b` < %s))",
		previousUpperBound[0].String(),
		previousUpperBound[0].String(),
		previousUpperBound[1].String(),
		upperBound[0].String(),
		upperBound[0].String(),
		upperBound[1].String()),
		chunk.String(),
	)

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound = upperBound
	upperBound = chunk.UpperBound.Value
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("((`a` > %s)\n OR (`a` = %s AND `b` >= %s)) AND ((`a` < %s)\n OR (`a` = %s AND `b` < %s))",
		previousUpperBound[0].String(),
		previousUpperBound[0].String(),
		previousUpperBound[1].String(),
		upperBound[0].String(),
		upperBound[0].String(),
		upperBound[1].String()),
		chunk.String(),
	)

	// Test it advances again
	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound = upperBound
	upperBound = chunk.UpperBound.Value
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("((`a` > %s)\n OR (`a` = %s AND `b` >= %s)) AND ((`a` < %s)\n OR (`a` = %s AND `b` < %s))",
		previousUpperBound[0].String(),
		previousUpperBound[0].String(),
		previousUpperBound[1].String(),
		upperBound[0].String(),
		upperBound[0].String(),
		upperBound[1].String()),
		chunk.String(),
	)

	// Repeat until done (final chunk is sent.)
	// Add to the total chunks
	totalChunks := 3 // 3 so far

	for i := 0; i < 5000; i++ {
		chunk, err = chunker.Next()
		if err != nil {
			break
		}
		totalChunks++
		assert.NotNil(t, chunk)
	}
	// there are 1001010 rows. It should be about 1002 chunks.
	// we don't care that it's exact, since we don't want a flaky
	// test if we make small changes.
	assert.True(t, totalChunks < 1005 && totalChunks > 995)
}

func TestCompositeChunkerBinary(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS composite_t1")
	runSQL(t, `CREATE TABLE composite_t1 (
		pk varbinary(40) NOT NULL,
		a int NOT NULL,
		b int NOT NULL,
		PRIMARY KEY (pk)
	)`)
	runSQL(t, `INSERT INTO composite_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM dual`)
	runSQL(t, `INSERT INTO composite_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM composite_t1 a JOIN composite_t1 b JOIN composite_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO composite_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM composite_t1 a JOIN composite_t1 b JOIN composite_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO composite_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM composite_t1 a JOIN composite_t1 b JOIN composite_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO composite_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM composite_t1 a JOIN composite_t1 b JOIN composite_t1 c LIMIT 1000000`)

	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "composite_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))

	// Assert that the types are correct.
	assert.Equal(t, []string{"varbinary"}, t1.keyColumnsMySQLTp)
	assert.Equal(t, binaryType, t1.keyDatums[0])

	chunker, err := NewChunker(t1, ChunkerDefaultTarget, logrus.New())
	assert.NoError(t, err)
	assert.IsType(t, &chunkerComposite{}, chunker)

	assert.NoError(t, chunker.Open())

	chunk, err := chunker.Next()
	assert.NoError(t, err)
	assert.NotContains(t, "`pk` >= ", chunk.String()) // first chunk is special
	upperBound := chunk.UpperBound.Value[0].String()

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound := upperBound
	upperBound = chunk.UpperBound.Value[0].String()
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("`pk` >= %s AND `pk` < %s", previousUpperBound, upperBound), chunk.String())

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound = upperBound
	upperBound = chunk.UpperBound.Value[0].String()
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("`pk` >= %s AND `pk` < %s", previousUpperBound, upperBound), chunk.String())

	// Test it advances again
	chunk, err = chunker.Next()
	assert.NoError(t, err)
	previousUpperBound = upperBound
	upperBound = chunk.UpperBound.Value[0].String()
	require.NotEqual(t, previousUpperBound, upperBound)
	assert.Equal(t, fmt.Sprintf("`pk` >= %s AND `pk` < %s", previousUpperBound, upperBound), chunk.String())

	// Repeat until done (final chunk is sent.)
	// Add to the total chunks
	totalChunks := 3 // 3 so far

	for i := 0; i < 5000; i++ {
		chunk, err = chunker.Next()
		if err != nil {
			break
		}
		totalChunks++
		assert.NotNil(t, chunk)
	}
	// there are 1001010 rows. It should be about 1002 chunks.
	// we don't care that it's exact, since we don't want a flaky
	// test if we make small changes.
	assert.True(t, totalChunks < 1005 && totalChunks > 995)
}
func TestCompositeChunkerInt(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS compositeint_t1")
	runSQL(t, `CREATE TABLE compositeint_t1 (
		pk int NOT NULL primary key auto_increment,
		a int NOT NULL,
		b int NOT NULL
	)`)
	runSQL(t, `INSERT INTO compositeint_t1 (pk, a, b) SELECT NULL, 1, 1 FROM dual`)
	runSQL(t, `INSERT INTO compositeint_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositeint_t1 a JOIN compositeint_t1 b JOIN compositeint_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO compositeint_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositeint_t1 a JOIN compositeint_t1 b JOIN compositeint_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO compositeint_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositeint_t1 a JOIN compositeint_t1 b JOIN compositeint_t1 c LIMIT 1000000`)
	runSQL(t, `INSERT INTO compositeint_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositeint_t1 a JOIN compositeint_t1 b JOIN compositeint_t1 c LIMIT 1000000`)
	// remove autoinc before discovery.
	runSQL(t, "ALTER TABLE compositeint_t1 CHANGE COLUMN pk pk int NOT NULL") //nolint: dupword

	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "compositeint_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))

	// Assert that the types are correct.
	assert.Equal(t, []string{"int"}, t1.keyColumnsMySQLTp)
	assert.Equal(t, signedType, t1.keyDatums[0])

	chunker, err := NewChunker(t1, ChunkerDefaultTarget, logrus.New())
	assert.NoError(t, err)
	assert.IsType(t, &chunkerComposite{}, chunker)

	assert.NoError(t, chunker.Open())

	// This might get messy if different versions skip
	// auto_inc values differently.

	chunk, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` < 1008", chunk.String()) // first chunk is special

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 1008 AND `pk` < 2032", chunk.String())

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2032 AND `pk` < 3033", chunk.String())

	totalChunks := 3 // 3 so far
	for i := 0; i < 5000; i++ {
		chunk, err = chunker.Next()
		if err != nil {
			break
		}
		totalChunks++
		assert.NotNil(t, chunk)
	}
	// there are 1001010 rows. It should be about 1002 chunks.
	// we don't care that it's exact, since we don't want a flaky
	// test if we make small changes.
	assert.True(t, totalChunks < 1005 && totalChunks > 995)
}

func TestCompositeLowWatermark(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS compositewatermark_t1")
	runSQL(t, `CREATE TABLE compositewatermark_t1 (
		pk int NOT NULL primary key auto_increment,
		a int NOT NULL,
		b int NOT NULL
	)`)
	runSQL(t, `INSERT INTO compositewatermark_t1 (pk, a, b) SELECT NULL, 1, 1 FROM dual`)
	runSQL(t, `INSERT INTO compositewatermark_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositewatermark_t1 a JOIN compositewatermark_t1 b JOIN compositewatermark_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO compositewatermark_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositewatermark_t1 a JOIN compositewatermark_t1 b JOIN compositewatermark_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO compositewatermark_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositewatermark_t1 a JOIN compositewatermark_t1 b JOIN compositewatermark_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO compositewatermark_t1 (pk, a, b) SELECT NULL, 1, 1 FROM compositewatermark_t1 a JOIN compositewatermark_t1 b JOIN compositewatermark_t1 c LIMIT 10000`)
	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "compositewatermark_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))

	chunker := &chunkerComposite{
		Ti:            t1,
		ChunkerTarget: ChunkerDefaultTarget,
		logger:        logrus.New(),
	}
	_, err = chunker.Next()
	assert.Error(t, err) // not open yet
	assert.NoError(t, chunker.Open())
	assert.Error(t, chunker.Open()) // double open should fail

	_, err = chunker.GetLowWatermark()
	assert.Error(t, err)

	assert.Equal(t, StartingChunkSize, int(chunker.chunkSize))
	chunk, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` < 1008", chunk.String()) // first chunk
	_, err = chunker.GetLowWatermark()
	assert.Error(t, err) // no feedback yet.
	chunker.Feedback(chunk, time.Millisecond*500)
	assert.Equal(t, StartingChunkSize, int(chunker.chunkSize)) // should not have changed yet (requires 10 feedbacks)

	_, err = chunker.GetLowWatermark()
	assert.Error(t, err) // there has been feedback, but watermark is not ready after first chunk.

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 1008 AND `pk` < 2032", chunk.String())
	chunker.Feedback(chunk, time.Second)
	assert.Equal(t, 100, int(chunker.chunkSize)) // usually requires 10 feedbacks, but changed because >5x target

	watermark, err := chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":1000,\"LowerBound\":{\"Value\": [\"1008\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2032\"],\"Inclusive\":false}}", watermark)

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2032 AND `pk` < 2133", chunk.String())
	chunker.Feedback(chunk, time.Second)
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":100,\"LowerBound\":{\"Value\": [\"2032\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2133\"],\"Inclusive\":false}}", watermark)

	chunkAsync1, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2133 AND `pk` < 2144", chunkAsync1.String())

	chunkAsync2, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2144 AND `pk` < 2155", chunkAsync2.String())

	chunkAsync3, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2155 AND `pk` < 2166", chunkAsync3.String())

	chunker.Feedback(chunkAsync2, time.Second)
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":100,\"LowerBound\":{\"Value\": [\"2032\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2133\"],\"Inclusive\":false}}", watermark)

	chunker.Feedback(chunkAsync3, time.Second)
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":100,\"LowerBound\":{\"Value\": [\"2032\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2133\"],\"Inclusive\":false}}", watermark)

	chunker.Feedback(chunkAsync1, time.Second)
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":10,\"LowerBound\":{\"Value\": [\"2155\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2166\"],\"Inclusive\":false}}", watermark)

	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "`pk` >= 2166 AND `pk` < 2177", chunk.String())
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":10,\"LowerBound\":{\"Value\": [\"2155\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2166\"],\"Inclusive\":false}}", watermark)
	chunker.Feedback(chunk, time.Second)
	watermark, err = chunker.GetLowWatermark()
	assert.NoError(t, err)
	assert.Equal(t, "{\"Key\":[\"pk\"],\"ChunkSize\":10,\"LowerBound\":{\"Value\": [\"2166\"],\"Inclusive\":true},\"UpperBound\":{\"Value\": [\"2177\"],\"Inclusive\":false}}", watermark)

	// Give enough feedback that the chunk size recalculation runs.
	assert.Equal(t, 10, int(chunker.chunkSize))
	for i := 0; i < 50; i++ {
		chunk, err = chunker.Next()
		assert.NoError(t, err)
		if chunk.ChunkSize != 10 {
			break // feedback has worked
		}
		chunker.Feedback(chunk, time.Millisecond*5) // say that it took 5ms to process 10 rows
	}
	assert.Len(t, chunker.chunkTimingInfo, 0)
	assert.Equal(t, 15, int(chunker.chunkSize)) // scales up a maximum of 50% at a time.
}

func TestCompositeSmallTable(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS compositesmall_t1")
	runSQL(t, `CREATE TABLE compositesmall_t1 (
		pk varbinary(40) NOT NULL,
		a int NOT NULL,
		b int NOT NULL,
		PRIMARY KEY (pk)
	)`)
	runSQL(t, `INSERT INTO compositesmall_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM dual`)
	runSQL(t, `INSERT INTO compositesmall_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM compositesmall_t1 a JOIN compositesmall_t1 b JOIN compositesmall_t1 c LIMIT 10`)
	runSQL(t, `INSERT INTO compositesmall_t1 (pk, a, b) SELECT UUID(), 1, 1 FROM compositesmall_t1 a JOIN compositesmall_t1 b JOIN compositesmall_t1 c LIMIT 10`)

	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "compositesmall_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))

	chunker, err := NewChunker(t1, ChunkerDefaultTarget, logrus.New())
	assert.NoError(t, err)
	assert.IsType(t, &chunkerComposite{}, chunker)

	assert.NoError(t, chunker.Open())

	chunk, err := chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "1=1", chunk.String()) // small chunk
	assert.NoError(t, chunker.Close())
}

func TestSetKey(t *testing.T) {
	runSQL(t, "DROP TABLE IF EXISTS setkey_t1")
	runSQL(t, `CREATE TABLE setkey_t1 (
		id INT NOT NULL PRIMARY KEY auto_increment,
		a int NOT NULL,
		b int NOT NULL,
		status ENUM('PENDING', 'ACTIVE', 'ARCHIVED') NOT NULL DEFAULT 'PENDING',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX s (status),
		INDEX u (updated_at),
		INDEX su (status, updated_at),
		INDEX ui (updated_at)
	)`)
	// 11K records.
	runSQL(t, `INSERT INTO setkey_t1 SELECT NULL, 1, 1, 'PENDING', NOW(), NOW() FROM dual`)
	runSQL(t, `INSERT INTO setkey_t1 SELECT NULL, 1, 1, 'PENDING', NOW(), NOW() FROM setkey_t1 a JOIN setkey_t1 b JOIN setkey_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO setkey_t1 SELECT NULL, 1, 1, 'PENDING', NOW(), NOW() FROM setkey_t1 a JOIN setkey_t1 b JOIN setkey_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO setkey_t1 SELECT NULL, 1, 1, 'PENDING', NOW(), NOW() FROM setkey_t1 a JOIN setkey_t1 b JOIN setkey_t1 c LIMIT 10000`)
	runSQL(t, `INSERT INTO setkey_t1 SELECT NULL, 1, 1, 'PENDING', NOW(), NOW() FROM setkey_t1 a JOIN setkey_t1 b JOIN setkey_t1 c LIMIT 10000`)

	db, err := sql.Open("mysql", dsn())
	assert.NoError(t, err)
	defer db.Close()

	t1 := NewTableInfo(db, "test", "setkey_t1")
	assert.NoError(t, t1.SetInfo(context.Background()))
	chunker := &chunkerComposite{
		Ti:            t1,
		ChunkerTarget: 100 * time.Millisecond,
		logger:        logrus.New(),
	}
	err = chunker.SetKey("s", "status = 'ARCHIVED' AND updated_at < NOW() - INTERVAL 1 DAY")
	assert.NoError(t, err)
	assert.NoError(t, chunker.Open())

	chunk, err := chunker.Next()
	assert.NoError(t, err)
	// Because there are zero rows with status archived or updated_at that old,
	// it returns 1 chunk with 1=1 and the original condition.
	assert.Equal(t, "1=1 AND (status = 'ARCHIVED' AND updated_at < NOW() - INTERVAL 1 DAY)", chunk.String())
	assert.NoError(t, chunker.Close())

	// If I reset again with a different condition it should range as chunks.
	chunker = &chunkerComposite{
		Ti:            t1,
		ChunkerTarget: 100 * time.Millisecond,
		logger:        logrus.New(),
	}
	err = chunker.SetKey("s", "status = 'PENDING' AND updated_at > NOW() - INTERVAL 1 DAY")
	assert.NoError(t, err)
	assert.NoError(t, chunker.Open())
	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "((`status` < \"PENDING\")\n OR (`status` = \"PENDING\" AND `id` < 1008)) AND (status = 'PENDING' AND updated_at > NOW() - INTERVAL 1 DAY)", chunk.String())

	// Check a chunk with both a lowerbound and upper bound.
	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "((`status` > \"PENDING\")\n OR (`status` = \"PENDING\" AND `id` >= 1008)) AND ((`status` < \"PENDING\")\n OR (`status` = \"PENDING\" AND `id` < 2032)) AND (status = 'PENDING' AND updated_at > NOW() - INTERVAL 1 DAY)", chunk.String())

	// repeat ~10 more times without calling Feedback()
	for i := 0; i < 8; i++ {
		_, err = chunker.Next()
		assert.NoError(t, err)
	}
	chunk, err = chunker.Next()
	assert.NoError(t, err)
	assert.Equal(t, "((`status` > \"PENDING\")\n OR (`status` = \"PENDING\" AND `id` >= 10040)) AND (status = 'PENDING' AND updated_at > NOW() - INTERVAL 1 DAY)", chunk.String())

	_, err = chunker.Next()
	assert.ErrorIs(t, err, ErrTableIsRead)

	assert.NoError(t, chunker.Close())

	// Test other index types.
	for _, index := range []string{"u", "su", "ui"} {
		chunker = &chunkerComposite{
			Ti:            t1,
			ChunkerTarget: 100 * time.Millisecond,
			logger:        logrus.New(),
		}
		err = chunker.SetKey(index, "updated_at < NOW() - INTERVAL 1 DAY")
		assert.NoError(t, err)
		assert.NoError(t, chunker.Open())
		chunk, err = chunker.Next()
		assert.NoError(t, err)
		assert.Equal(t, "1=1 AND (updated_at < NOW() - INTERVAL 1 DAY)", chunk.String())

		// check the key parts are correct.
		switch index {
		case "u":
			assert.Equal(t, []string{"updated_at", "id"}, chunker.chunkKeys)
		case "su":
			assert.Equal(t, []string{"status", "updated_at", "id"}, chunker.chunkKeys)
		case "ui":
			assert.Equal(t, []string{"updated_at", "id"}, chunker.chunkKeys)
		}
		assert.NoError(t, chunker.Close())
	}
}
