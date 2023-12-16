package main

import (
  "hash/fnv"
)

// Hash function to create a consistent hash of the metric name
func hash(s string) uint32 {
  h := fnv.New32a()
  h.Write([]byte(s))
  return h.Sum32()
}
