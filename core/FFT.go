package core

import (
	"math"
)

func FFT(input []float64) []complex128 {
	complexArray := make([]complex128, len(input))
	for i, v := range input {
		complexArray[i] = complex(v, 0)
	}

	fftResult := make([]complex128, len(complexArray))
	copy(fftResult, complexArray)
	return recursiveFFT(fftResult)
}

func recursiveFFT(complexArray []complex128) []complex128 {
	N := len(complexArray)
	if N <= 1 {
		return complexArray
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
		t := complex(math.Cos(-2*math.Pi*float64(k)/float64(N)), math.Sin(-2*math.Pi*float64(k)/float64(N)))
		fftResult[k] = even[k] + t*odd[k]
		fftResult[k+N/2] = even[k] - t*odd[k]
	}

	return fftResult
}

/*
Package core provides digital signal processing functionality for audio analysis and fingerprinting.
This file implements the Fast Fourier Transform (FFT), a fundamental algorithm for converting
audio signals from the time domain to the frequency domain.

The Fourier Transform is based on the principle that any periodic signal can be decomposed into
a sum of sine and cosine waves at different frequencies. While the Discrete Fourier Transform (DFT)
achieves this transformation, it has a computational complexity of O(N²), making it impractical
for large audio samples. The FFT algorithm, discovered by Cooley and Tukey in 1965, reduces this
complexity to O(N log N) through a divide-and-conquer approach.

How FFT Works:
The algorithm recursively splits the input signal into even-indexed and odd-indexed samples,
computes the FFT of each half, and then combines the results using complex number arithmetic.
This splitting continues until we reach base cases of single elements, which are trivially
already in the frequency domain.

The combination step (known as the "butterfly operation") uses twiddle factors - complex numbers
of the form e^(-2πik/N) that represent rotations in the complex plane. These rotations are necessary
because the odd-indexed samples need to be phase-shifted before being combined with the even-indexed
samples to produce the correct frequency components.

Mathematical Foundation:
For a signal x[n] of length N, the DFT is defined as:
  X[k] = Σ(n=0 to N-1) x[n] · e^(-2πikn/N)

The FFT exploits the symmetry property that splitting this into even and odd indices yields:
  X[k] = E[k] + W^k · O[k]           for k = 0 to N/2-1
  X[k+N/2] = E[k] - W^k · O[k]       for k = 0 to N/2-1

where E[k] is the FFT of even samples, O[k] is the FFT of odd samples, and W^k = e^(-2πik/N)
is the twiddle factor.

Applications in Audio Processing:
The FFT is crucial for audio fingerprinting systems like Shazam because it reveals which
frequencies are present in a signal at any given time. By applying FFT to overlapping windows
of audio samples (creating a spectrogram), we can identify characteristic frequency patterns
that uniquely identify a song, even in the presence of noise.

For a visual explanation of the FFT algorithm, see: https://www.youtube.com/watch?v=spUNpyF58BY

Implementation Notes:
- Input is converted from real numbers to complex numbers (with zero imaginary parts)
- The algorithm requires input length to be a power of 2 for optimal performance
- Twiddle factors are computed using Euler's formula: e^(iθ) = cos(θ) + i·sin(θ)
- The output is an array of complex numbers representing frequency components
*/
