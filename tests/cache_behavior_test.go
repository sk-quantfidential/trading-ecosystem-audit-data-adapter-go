package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// CacheBehaviorTestSuite tests the behavior of cache repository operations
type CacheBehaviorTestSuite struct {
	BehaviorTestSuite
}

// TestCacheBehaviorSuite runs the cache behavior test suite
func TestCacheBehaviorSuite(t *testing.T) {
	suite.Run(t, new(CacheBehaviorTestSuite))
}

// TestCacheBasicOperations tests basic cache operations
func (suite *CacheBehaviorTestSuite) TestCacheBasicOperations() {
	suite.T().Log("Testing cache basic operations")
	suite.RunScenario("cache_operations")
}

// TestCacheStringOperations tests string value caching
func (suite *CacheBehaviorTestSuite) TestCacheStringOperations() {
	var (
		key   = "test:string:" + GenerateTestID("key")
		value = "Hello, Cache World!"
		ttl   = 5 * time.Minute
	)

	suite.Given("a string value to cache", func() {
		// Value defined above
	}).When("storing the string in cache", func() {
		err := suite.adapter.Set(suite.ctx, key, value, ttl)
		suite.Require().NoError(err)
	}).Then("the string should be retrievable", func() {
		var retrieved string
		err := suite.adapter.Get(suite.ctx, key, &retrieved)
		suite.Require().NoError(err)
		suite.Equal(value, retrieved)
	}).And("the key should exist", func() {
		exists, err := suite.adapter.Exists(suite.ctx, key)
		suite.Require().NoError(err)
		suite.True(exists)
	})
}

// TestCacheComplexDataTypes tests caching of complex data structures
func (suite *CacheBehaviorTestSuite) TestCacheComplexDataTypes() {
	var (
		key = "test:complex:" + GenerateTestID("key")
		ttl = 10 * time.Minute
	)

	suite.Given("complex data structures to cache", func() {
		// Test struct-like data
		complexData := map[string]interface{}{
			"id":        "user-123",
			"name":      "John Doe",
			"email":     "john@example.com",
			"age":       30,
			"active":    true,
			"tags":      []string{"premium", "verified"},
			"metadata": map[string]interface{}{
				"last_login":    "2023-10-01T10:00:00Z",
				"login_count":   42,
				"preferences": map[string]interface{}{
					"theme":        "dark",
					"notifications": true,
				},
			},
		}

		err := suite.adapter.Set(suite.ctx, key, complexData, ttl)
		suite.Require().NoError(err)
	}).When("retrieving the complex data", func() {
		var retrieved map[string]interface{}
		err := suite.adapter.Get(suite.ctx, key, &retrieved)
		suite.Require().NoError(err)

		suite.Then("all data should be preserved", func() {
			suite.Equal("user-123", retrieved["id"])
			suite.Equal("John Doe", retrieved["name"])
			suite.Equal("john@example.com", retrieved["email"])
			suite.Equal(float64(30), retrieved["age"]) // JSON numbers become float64
			suite.Equal(true, retrieved["active"])

			// Check array
			tags, ok := retrieved["tags"].([]interface{})
			suite.True(ok, "Tags should be an array")
			suite.Len(tags, 2)
			suite.Equal("premium", tags[0])
			suite.Equal("verified", tags[1])

			// Check nested object
			metadata, ok := retrieved["metadata"].(map[string]interface{})
			suite.True(ok, "Metadata should be an object")
			suite.Equal("2023-10-01T10:00:00Z", metadata["last_login"])
			suite.Equal(float64(42), metadata["login_count"])

			// Check deeply nested object
			preferences, ok := metadata["preferences"].(map[string]interface{})
			suite.True(ok, "Preferences should be an object")
			suite.Equal("dark", preferences["theme"])
			suite.Equal(true, preferences["notifications"])
		})
	})
}

// TestCacheTTLBehavior tests TTL (time-to-live) functionality
func (suite *CacheBehaviorTestSuite) TestCacheTTLBehavior() {
	var (
		shortTTLKey = "test:ttl:short:" + GenerateTestID("key")
		longTTLKey  = "test:ttl:long:" + GenerateTestID("key")
		value       = "TTL Test Value"
		shortTTL    = 2 * time.Second
		longTTL     = 10 * time.Minute
	)

	suite.Given("keys with different TTL values", func() {
		// Set key with short TTL
		err := suite.adapter.Set(suite.ctx, shortTTLKey, value, shortTTL)
		suite.Require().NoError(err)

		// Set key with long TTL
		err = suite.adapter.Set(suite.ctx, longTTLKey, value, longTTL)
		suite.Require().NoError(err)
	}).When("checking TTL immediately after setting", func() {
		shortTTLRemaining, err := suite.adapter.GetTTL(suite.ctx, shortTTLKey)
		suite.Require().NoError(err)

		longTTLRemaining, err := suite.adapter.GetTTL(suite.ctx, longTTLKey)
		suite.Require().NoError(err)

		suite.Then("TTL values should be approximately correct", func() {
			suite.Greater(shortTTLRemaining, 1*time.Second, "Short TTL should be > 1 second")
			suite.LessOrEqual(shortTTLRemaining, shortTTL, "Short TTL should be <= set value")

			suite.Greater(longTTLRemaining, 9*time.Minute, "Long TTL should be > 9 minutes")
			suite.LessOrEqual(longTTLRemaining, longTTL, "Long TTL should be <= set value")
		})
	}).And("after waiting for short TTL to expire", func() {
		// Wait for short TTL to expire (add buffer for processing time)
		time.Sleep(shortTTL + 500*time.Millisecond)

		shortExists, err := suite.adapter.Exists(suite.ctx, shortTTLKey)
		suite.Require().NoError(err)

		longExists, err := suite.adapter.Exists(suite.ctx, longTTLKey)
		suite.Require().NoError(err)

		suite.Then("short TTL key should be expired", func() {
			suite.False(shortExists, "Short TTL key should be expired")
		}).And("long TTL key should still exist", func() {
			suite.True(longExists, "Long TTL key should still exist")
		})
	})
}

// TestCacheBulkOperations tests bulk cache operations
func (suite *CacheBehaviorTestSuite) TestCacheBulkOperations() {
	var (
		keyPrefix = "test:bulk:" + GenerateTestID("prefix")
		ttl       = 5 * time.Minute
		items     = make(map[string]interface{})
		keys      []string
	)

	suite.Given("multiple items for bulk operations", func() {
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("%s:item-%d", keyPrefix, i)
			value := map[string]interface{}{
				"id":    i,
				"name":  fmt.Sprintf("Item %d", i),
				"value": i * 10,
			}
			items[key] = value
			keys = append(keys, key)
		}
	}).When("storing items in bulk", func() {
		err := suite.adapter.SetMany(suite.ctx, items, ttl)
		suite.Require().NoError(err)
	}).Then("all items should be retrievable individually", func() {
		for key, expectedValue := range items {
			var retrieved map[string]interface{}
			err := suite.adapter.Get(suite.ctx, key, &retrieved)
			suite.Require().NoError(err)

			expected := expectedValue.(map[string]interface{})
			suite.Equal(float64(expected["id"].(int)), retrieved["id"])
			suite.Equal(expected["name"], retrieved["name"])
			suite.Equal(float64(expected["value"].(int)), retrieved["value"])
		}
	}).And("items should be retrievable in bulk", func() {
		retrievedItems, err := suite.adapter.GetMany(suite.ctx, keys)
		suite.Require().NoError(err)

		suite.Len(retrievedItems, len(keys), "Should retrieve all keys")

		for _, key := range keys {
			suite.Contains(retrievedItems, key, "Retrieved items should contain key %s", key)
		}
	}).And("items can be deleted in bulk", func() {
		err := suite.adapter.DeleteMany(suite.ctx, keys)
		suite.Require().NoError(err)

		// Verify deletion
		for _, key := range keys {
			exists, err := suite.adapter.Exists(suite.ctx, key)
			suite.Require().NoError(err)
			suite.False(exists, "Key %s should be deleted", key)
		}
	})
}

// TestCachePatternOperations tests pattern-based operations
func (suite *CacheBehaviorTestSuite) TestCachePatternOperations() {
	var (
		baseKey = "test:pattern:" + GenerateTestID("base")
		ttl     = 5 * time.Minute
		keys    []string
	)

	suite.Given("keys matching and not matching a pattern", func() {
		// Create keys that match pattern
		matchingKeys := []string{
			baseKey + ":user:1",
			baseKey + ":user:2",
			baseKey + ":user:3",
		}

		// Create keys that don't match pattern
		nonMatchingKeys := []string{
			baseKey + ":admin:1",
			baseKey + ":guest:1",
			"different:pattern:key",
		}

		allKeys := append(matchingKeys, nonMatchingKeys...)

		for _, key := range allKeys {
			value := fmt.Sprintf("Value for %s", key)
			err := suite.adapter.Set(suite.ctx, key, value, ttl)
			suite.Require().NoError(err)
			keys = append(keys, key)
		}
	}).When("searching for keys by pattern", func() {
		pattern := baseKey + ":user:*"
		foundKeys, err := suite.adapter.GetKeysByPattern(suite.ctx, pattern)
		suite.Require().NoError(err)

		suite.Then("only matching keys should be found", func() {
			suite.Len(foundKeys, 3, "Should find exactly 3 matching keys")

			expectedKeys := map[string]bool{
				baseKey + ":user:1": true,
				baseKey + ":user:2": true,
				baseKey + ":user:3": true,
			}

			for _, key := range foundKeys {
				suite.True(expectedKeys[key], "Found key %s should be expected", key)
			}
		})
	}).And("when deleting keys by pattern", func() {
		pattern := baseKey + ":user:*"
		deletedCount, err := suite.adapter.DeleteByPattern(suite.ctx, pattern)
		suite.Require().NoError(err)

		suite.Then("matching keys should be deleted", func() {
			suite.Equal(int64(3), deletedCount, "Should delete exactly 3 keys")

			// Verify matching keys are deleted
			userKeys := []string{
				baseKey + ":user:1",
				baseKey + ":user:2",
				baseKey + ":user:3",
			}

			for _, key := range userKeys {
				exists, err := suite.adapter.Exists(suite.ctx, key)
				suite.Require().NoError(err)
				suite.False(exists, "User key %s should be deleted", key)
			}

			// Verify non-matching keys still exist
			nonUserKeys := []string{
				baseKey + ":admin:1",
				baseKey + ":guest:1",
			}

			for _, key := range nonUserKeys {
				exists, err := suite.adapter.Exists(suite.ctx, key)
				suite.Require().NoError(err)
				suite.True(exists, "Non-user key %s should still exist", key)
			}
		})
	})

	// Cleanup remaining keys
	suite.TearDownTest()
}

// TestCacheTTLManagement tests TTL management operations
func (suite *CacheBehaviorTestSuite) TestCacheTTLManagement() {
	var (
		key       = "test:ttl:management:" + GenerateTestID("key")
		value     = "TTL Management Test"
		initialTTL = 5 * time.Minute
		newTTL     = 10 * time.Minute
	)

	suite.Given("a cached item with initial TTL", func() {
		err := suite.adapter.Set(suite.ctx, key, value, initialTTL)
		suite.Require().NoError(err)
	}).When("updating the TTL", func() {
		err := suite.adapter.SetTTL(suite.ctx, key, newTTL)
		suite.Require().NoError(err)
	}).Then("the new TTL should be set", func() {
		currentTTL, err := suite.adapter.GetTTL(suite.ctx, key)
		suite.Require().NoError(err)

		suite.Greater(currentTTL, 9*time.Minute, "TTL should be greater than 9 minutes")
		suite.LessOrEqual(currentTTL, newTTL, "TTL should not exceed new TTL value")
	}).And("the value should still be accessible", func() {
		var retrieved string
		err := suite.adapter.Get(suite.ctx, key, &retrieved)
		suite.Require().NoError(err)
		suite.Equal(value, retrieved)
	})
}

// TestCacheHealthAndStats tests cache health and statistics
func (suite *CacheBehaviorTestSuite) TestCacheHealthAndStats() {
	suite.Given("a functioning cache", func() {
		// Set up some test data
		testKey := "test:health:" + GenerateTestID("key")
		testValue := "Health Test Value"
		err := suite.adapter.Set(suite.ctx, testKey, testValue, time.Minute)
		suite.Require().NoError(err)
	}).When("checking cache health", func() {
		err := suite.adapter.Ping(suite.ctx)
		suite.Require().NoError(err)

		suite.Then("cache should be healthy", func() {
			// Health check passed
		})
	}).And("when getting cache statistics", func() {
		stats, err := suite.adapter.GetStats(suite.ctx)
		suite.Require().NoError(err)

		suite.Then("statistics should be available", func() {
			suite.NotNil(stats, "Statistics should not be nil")
			suite.NotEmpty(stats, "Statistics should not be empty")

			// Basic check that we get some stats back
			// The exact stats depend on Redis implementation
			suite.T().Logf("Cache stats: %+v", stats)
		})
	})
}

// TestCacheErrorHandling tests cache error scenarios
func (suite *CacheBehaviorTestSuite) TestCacheErrorHandling() {
	var nonExistentKey = "test:error:nonexistent:" + GenerateTestID("key")

	suite.Given("a non-existent cache key", func() {
		// Ensure key doesn't exist
		exists, err := suite.adapter.Exists(suite.ctx, nonExistentKey)
		suite.Require().NoError(err)
		suite.False(exists)
	}).When("trying to get the non-existent key", func() {
		var value string
		err := suite.adapter.Get(suite.ctx, nonExistentKey, &value)

		suite.Then("an error should be returned", func() {
			suite.Error(err, "Should return error for non-existent key")
		})
	}).And("when trying to get TTL for non-existent key", func() {
		_, err := suite.adapter.GetTTL(suite.ctx, nonExistentKey)

		suite.Then("an error should be returned", func() {
			suite.Error(err, "Should return error for non-existent key TTL")
		})
	})
}

// TestCachePerformance tests cache performance characteristics
func (suite *CacheBehaviorTestSuite) TestCachePerformance() {
	var (
		keyPrefix = "test:performance:" + GenerateTestID("prefix")
		value     = "Performance Test Value"
		ttl       = 5 * time.Minute
		keys      []string
	)

	suite.Given("performance test configuration", func() {
		// Skip if in CI to avoid flaky tests
		if IsCI() {
			suite.T().Skip("Skipping performance test in CI environment")
		}
	}).When("performing many cache operations", func() {
		suite.AssertPerformance("set 100 cache items", 2*time.Second, func() {
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("%s:item-%d", keyPrefix, i)
				err := suite.adapter.Set(suite.ctx, key, value, ttl)
				suite.Require().NoError(err)
				keys = append(keys, key)
			}
		})
	}).Then("retrieval performance should be acceptable", func() {
		suite.AssertPerformance("get 100 cache items", 1*time.Second, func() {
			for _, key := range keys {
				var retrieved string
				err := suite.adapter.Get(suite.ctx, key, &retrieved)
				suite.Require().NoError(err)
			}
		})
	}).And("bulk operations should be faster", func() {
		items := make(map[string]interface{})
		for i := 100; i < 200; i++ {
			key := fmt.Sprintf("%s:bulk-item-%d", keyPrefix, i)
			items[key] = value
		}

		suite.AssertPerformance("bulk set 100 items", 1*time.Second, func() {
			err := suite.adapter.SetMany(suite.ctx, items, ttl)
			suite.Require().NoError(err)
		})

		bulkKeys := make([]string, 0, len(items))
		for key := range items {
			bulkKeys = append(bulkKeys, key)
			keys = append(keys, key) // For cleanup
		}

		suite.AssertPerformance("bulk get 100 items", 500*time.Millisecond, func() {
			_, err := suite.adapter.GetMany(suite.ctx, bulkKeys)
			suite.Require().NoError(err)
		})
	})

	// Cleanup
	if len(keys) > 0 {
		_ = suite.adapter.DeleteMany(suite.ctx, keys)
	}
}