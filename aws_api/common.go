package aws_api

import (
	"fmt"
	"log"
	"reflect" // Import the reflect package
	"time"

	"github.com/AlexeyBeley/go_common/logger"
)

var lg = &(logger.Logger{})

// Retry executes a generic function and retries it up to maxRetries times if it returns an error.
// The generic function must return an error as its last return value.
// It can accept any number of arguments.
func Retry(maxRetries int, delay time.Duration, fn interface{}, args ...interface{}) (reflect.Value, error) {
	// 1. Validate the provided function
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Ensure fn is actually a function
	if fnType.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("Retry: fn must be a function, got %T", fn)
	}

	// Ensure the function's last return value is an error
	if fnType.NumOut() == 0 || fnType.Out(fnType.NumOut()-1).String() != "error" {
		return reflect.Value{}, fmt.Errorf("Retry: fn must return an error as its last return value")
	}

	// 2. Prepare arguments for the function call
	// Convert provided args to reflect.Value
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// Basic argument type validation (can be more robust)
	if fnType.NumIn() != len(in) {
		return reflect.Value{}, fmt.Errorf("Retry: function expects %d arguments, but received %d", fnType.NumIn(), len(in))
	}
	for i := 0; i < fnType.NumIn(); i++ {
		if !in[i].Type().AssignableTo(fnType.In(i)) {
			return reflect.Value{}, fmt.Errorf("Retry: argument %d type mismatch: expected %s, got %s", i, fnType.In(i), in[i].Type())
		}
	}

	// 3. Retry Loop
	var lastErr error
	var resultValue reflect.Value // To store the first non-error return value

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			fmt.Printf("Retry attempt %d/%d after %v delay...\n", i, maxRetries, delay)
			time.Sleep(delay)
		} else {
			fmt.Printf("Initial attempt...\n")
		}

		// Call the function using reflection
		out := fnValue.Call(in)

		// Check the last return value for an error
		if len(out) > 0 {
			lastOut := out[len(out)-1] // The last return value
			if lastOut.IsValid() && !lastOut.IsNil() {
				// The last return value is an error
				lastErr = lastOut.Interface().(error)
				fmt.Printf("Attempt %d failed: %v\n", i+1, lastErr)
				continue // Continue to next retry
			}
		}

		// If we reached here, the function did not return an error (or no error return value)
		// Store the first return value (if any) and succeed.
		if len(out) > 0 {
			resultValue = out[0] // Assuming the first return value is the main result
		} else {
			resultValue = reflect.Value{} // No return value
		}
		return resultValue, nil // Success!
	}

	// If the loop finishes, all retries failed
	return reflect.Value{}, fmt.Errorf("Retry: all %d attempts failed. Last error: %w", maxRetries+1, lastErr)
}

// --- Example Generic Functions to Test ---

// Example 1: A function that sometimes fails and returns a string
var callCount1 int

func unreliableServiceCall(input string, attempt int) (string, error) {
	callCount1++
	fmt.Printf("  UnreliableServiceCall: Attempt %d, Input: %s\n", attempt, input)
	if callCount1 < 3 { // Fail first 2 attempts
		return "", fmt.Errorf("simulated network error on attempt %d", callCount1)
	}
	return "Data from " + input, nil
}

// Example 2: A function that always succeeds and returns an int
func stableCalculation(a, b int) (int, error) {
	fmt.Printf("  StableCalculation: Calculating %d + %d\n", a, b)
	return a + b, nil
}

// Example 3: A function with no return value other than error
var callCount3 int

func performAction(action string) error {
	callCount3++
	fmt.Printf("  PerformAction: Executing '%s', Attempt %d\n", action, callCount3)
	if callCount3 < 2 { // Fail first attempt
		return fmt.Errorf("simulated action failure on attempt %d", callCount3)
	}
	return nil
}

// Example 4: A function with multiple return values
var callCount4 int

func fetchData(url string) (string, int, error) {
	callCount4++
	fmt.Printf("  FetchData: Fetching from %s, Attempt %d\n", url, callCount4)
	if callCount4 < 2 {
		return "", 0, fmt.Errorf("simulated fetch error")
	}
	return "Fetched content", 200, nil
}

func main() {
	log.Println("--- Test Case 1: Unreliable Service Call (succeeds on 3rd retry) ---")
	callCount1 = 0                                                                 // Reset counter for test
	result1, err := Retry(3, 1*time.Second, unreliableServiceCall, "user_data", 1) // Pass '1' as dummy attempt arg
	if err != nil {
		log.Printf("Test Case 1 failed: %v", err)
	} else {
		fmt.Printf("Test Case 1 Succeeded. Result: %s\n", result1.String())
	}

	log.Println("\n--- Test Case 2: Stable Calculation (succeeds immediately) ---")
	result2, err := Retry(3, 1*time.Second, stableCalculation, 10, 20)
	if err != nil {
		log.Printf("Test Case 2 failed: %v", err)
	} else {
		fmt.Printf("Test Case 2 Succeeded. Result: %d\n", result2.Int())
	}

	log.Println("\n--- Test Case 3: Perform Action (succeeds on 2nd retry) ---")
	callCount3 = 0 // Reset counter for test
	_, err = Retry(3, 500*time.Millisecond, performAction, "cleanup_task")
	if err != nil {
		log.Printf("Test Case 3 failed: %v", err)
	} else {
		fmt.Printf("Test Case 3 Succeeded.\n")
	}

	log.Println("\n--- Test Case 4: Function with Multiple Return Values ---")
	callCount4 = 0 // Reset counter for test
	result4, err := Retry(1, 1*time.Second, fetchData, "http://api.example.com/data")
	if err != nil {
		log.Printf("Test Case 4 failed: %v", err)
	} else {
		// Access multiple return values from the 'out' slice if needed
		// For simplicity, Retry only returns the first value.
		// In a real scenario, you might return the entire 'out' slice or a custom struct.
		fmt.Printf("Test Case 4 Succeeded. Content: %s, Status: %d\n", result4.String(), 1) // Accessing out[1] here is problematic as 'out' is local.
		// To get all results, you'd need to modify Retry to return []reflect.Value
	}

	log.Println("\n--- Test Case 5: Always Failing Function (all retries fail) ---")
	callCount1 = 0                                                                     // Reset counter for test
	result5, err := Retry(2, 1*time.Second, unreliableServiceCall, "critical_data", 1) // Only 2 retries (3 attempts total)
	if err != nil {
		log.Printf("Test Case 5 failed as expected: %v", err)
	} else {
		fmt.Printf("Test Case 5 Succeeded (unexpected): %s\n", result5.String())
	}
}
