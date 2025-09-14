package main

import (
	"math"
)

// FFT computes the Fast Fourier Transform with optimizations
func FFT(input []float64) []complex128 {
	n := len(input)
	
	// Convert to complex and ensure power of 2
	complexArray := make([]complex128, n)
	for i, v := range input {
		complexArray[i] = complex(v, 0)
	}

	return recursiveFFT(complexArray)
}

func recursiveFFT(complexArray []complex128) []complex128 {
	N := len(complexArray)
	if N <= 1 {
		return complexArray
	}

	// Handle non-power-of-2 sizes
	if N&(N-1) != 0 {
		return dftFallback(complexArray)
	}

	even := make([]complex128, N/2)
	odd := make([]complex128, N/2)
	for i := 0; i < N/2; i++ {
		even[i] = complexArray[2*i]
		odd[i] = complexArray[2*i+1]
	}

	even = recursiveFFT(even)
	odd = recursiveFFT(odd)

	fftResult := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		angle := -2 * math.Pi * float64(k) / float64(N)
		t := complex(math.Cos(angle), math.Sin(angle)) * odd[k]
		fftResult[k] = even[k] + t
		fftResult[k+N/2] = even[k] - t
	}

	return fftResult
}

// Fallback DFT for non-power-of-2 sizes
func dftFallback(input []complex128) []complex128 {
	N := len(input)
	output := make([]complex128, N)

	for k := 0; k < N; k++ {
		sum := complex(0, 0)
		for n := 0; n < N; n++ {
			angle := -2 * math.Pi * float64(k) * float64(n) / float64(N)
			w := complex(math.Cos(angle), math.Sin(angle))
			sum += input[n] * w
		}
		output[k] = sum
	}

	return output
}
