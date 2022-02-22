package operations

func int32Ptr(i int) *int32 {
	i32 := int32(i)
	return &i32
}

func strPtr(s string) *string {
	return &s
}
