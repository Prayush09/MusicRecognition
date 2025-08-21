# Audio Fingerprinting Algorithm

## Project Overview
Implementation of an audio fingerprinting system similar to Shazam's algorithm for song identification. This project captures audio, processes it through frequency domain analysis, and generates unique acoustic fingerprints for song recognition.

## The Fingerprinting Process
- **Convert to frequency domain** (FFT step)
- **Find spectral peaks** - the strongest frequencies at each time slice
- **Create constellation map** - plot these peaks across time and frequency
- **Generate hash pairs** - create unique identifiers from peak patterns
- **Store in database** - millions of songs become millions of compact fingerprints

## Implementation Steps

### 1. Audio Recording and Storage ✅
**Process**: Initialize() → Open Stream ✅ → Start() ✅ → Read data in loop → Stop() → Close() → Terminate() ✅

**Status**: COMPLETED

**Output**: Raw audio data stored in time-domain array slice (`[]int16`)

### 2. Time-Domain to Frequency-Domain Conversion 🔄
**Current Stage**: Our array now contains audio data in the time-domain form

**Next Step**: Convert from time-domain to frequency-domain using Discrete Fourier Transform (FFT)

---

# Audio Chunking Strategy for Frequency Domain Conversion

## Overview
This document outlines the rationale behind dividing the time-domain audio data into chunks of `TOTAL_SAMPLES / 256` for FFT processing in our acoustic fingerprinting system.

## Problem Statement
The recorded time-domain audio array contains a large dataset (220,500 samples for 5 seconds at 44.1kHz sample rate). Direct FFT processing of the entire dataset would result in poor time resolution for fingerprinting applications.

## Solution: Chunked Processing

### Chunk Size Calculation
- **Total samples**: 220,500 (5 seconds × 44,100 Hz)
- **Division factor**: 256
- **Resulting chunk size**: ~861 samples per chunk
- **Number of chunks**: 256 chunks
- **Time per chunk**: ~19.5 milliseconds

## Technical Rationale

### 1. Time-Frequency Resolution Trade-off
- **Larger chunks**: Better frequency resolution, worse time resolution
- **Smaller chunks**: Better time resolution, worse frequency resolution
- **Our choice (256 division)**: Balanced approach maintaining both adequate frequency resolution and time tracking

### 2. Frequency Resolution Analysis
- **Frequency resolution**: 44,100 Hz ÷ 861 samples = ~51 Hz per frequency bin
- **Significance**: 51 Hz resolution is sufficient to distinguish between musical notes and harmonics
- **Critical for fingerprinting**: Accurate frequency localization of spectral peaks is essential for reliable acoustic fingerprints

### 3. FFT Efficiency Considerations
- **Algorithm preference**: FFT performs optimally with power-of-2 input sizes (O(n log n) complexity)
- **Current chunk size**: 861 samples (not a power of 2)
- **Optimization option**: Zero-padding to 1024 samples for improved FFT performance

### 4. Temporal Resolution Benefits
- **Analysis windows**: 256 chunks provide adequate temporal sampling
- **Time tracking**: ~19.5ms windows enable tracking of rapid spectral changes
- **Fingerprinting requirement**: Sufficient temporal resolution for peak constellation mapping

## Signal Processing Pipeline Integration

This chunking strategy serves as the foundation for:

1. **FFT Conversion**: Time-domain → Frequency-domain transformation
2. **Spectrogram Generation**: Time-frequency plot creation
3. **Spectral Peak Detection**: Identification of prominent frequency components
4. **Constellation Mapping**: Peak plotting across time-frequency space
5. **Acoustic Fingerprinting**: Unique song signature generation

## Implementation Notes

### Data Format
- **Input**: Time-domain audio samples ([]int16)
- **Processing**: Individual chunks of ~861 samples each
- **Output**: Frequency-domain representation per chunk

### Performance Considerations
- Consider zero-padding chunks to 1024 samples for optimal FFT performance
- Maintain consistent chunk overlap if implementing sliding window analysis
- Monitor memory usage for large audio files

## Key Terminology
- **Spectral peaks**: Strongest frequency components in each time window
- **Frequency resolution**: Precision of frequency measurement (Hz per bin)
- **Spectrogram**: Time-frequency representation of audio signal
- **Constellation map**: Plot of spectral peaks across time-frequency space
- **Acoustic fingerprinting**: Process of creating unique audio signatures

## Validation Criteria
- Frequency resolution adequate for musical note discrimination
- Temporal resolution sufficient for tracking audio events
- Chunk size compatible with FFT processing requirements
- Output suitable for downstream fingerprinting algorithms


# FFT Algorithm Deep Dive

## Overview
This document provides a comprehensive analysis of the Fast Fourier Transform (FFT) algorithms used in our audio fingerprinting system, with particular focus on the Bluestein algorithm and its recursive structure.

## Mathematical Foundation

### Discrete Fourier Transform (DFT)
The standard DFT formula:
```
DFT[k] = Σ(n=0 to N-1) x[n] * e^(-2πi*nk/N)
```

### Bluestein's Algorithm Identity
Bluestein discovered that the DFT can be mathematically rearranged into convolution form:

**Original DFT:**
```
DFT[k] = Σ(n=0 to N-1) x[n] * e^(-2πi*nk/N)
```

**Bluestein's Transformation:**
```
DFT[k] = e^(-πi*k²/N) * Σ(n=0 to N-1) [x[n] * e^(-πi*n²/N)] * e^(πi*(k-n)²/N)
```

**Mathematical Proof:**
When multiplying the outer exponential into the summation and applying exponent rules:
- Expand: `(k-n)² = k² - 2kn + n²`
- Simplify exponents: `-πi*n²/N - πi*k²/N + πi*(k² - 2kn + n²)/N = -2πi*kn/N`
- Result: Original DFT formula ✓

**Key Insight:** This rearrangement converts DFT computation into a convolution problem, which can be solved efficiently using power-of-2 FFTs.

## FFT Implementation Architecture

### Algorithm Decision Tree
```
FFTReal(float64_data) 
├─ Convert to complex128
└─ FFT(complex_data)
   ├─ If length is power-of-2 → radix2FFT (Cooley-Tukey)
   └─ If length is arbitrary → bluesteinFFT
```

### For Our 860-Sample Chunks
Since 860 is not a power of 2, the system uses Bluestein's algorithm.

## Bluestein Algorithm Execution Flow

### Step 1: Setup and Padding
```go
lx := 860  // Original chunk size
// Calculate padding: NextPowerOf2(860*2-1) = NextPowerOf2(1719) = 2048
a := ZeroPad(data, 2048)  // Pad 860 → 2048 samples
```

### Step 2: Chirp Factor Generation
**Chirp Factors:** Complex exponentials with quadratic phase progression
```go
factors[i] = e^(+πi*i²/860) = cos(π*i²/860) + i*sin(π*i²/860)      // Normal chirp
invFactors[i] = e^(-πi*i²/860) = cos(π*i²/860) - i*sin(π*i²/860)  // Inverse chirp
```

**Why "Chirp":** The term π*i²/860 is quadratic in i, creating a frequency that increases as i² - similar to a bird's chirp sound that sweeps from low to high frequency.

### Step 3: Pre-multiplication
```go
a[n] = x[n] * invFactors[n]  // Apply e^(-πi*n²/860)
```
**Purpose:** Transforms input data according to Bluestein's mathematical identity, preparing it for convolution.

### Step 4: Chirp Kernel Setup
```go
b[i] = factors[i]        // e^(πi*i²/860)
b[2048-i] = factors[i]   // Mirror for circular convolution
```

### Step 5: Convolution via Frequency Domain
**The Recursive Call Structure:**
```go
Convolve(a, b):  // Both length 2048
├─ fft_x = FFT(a)  → radix2FFT (2048 is power-of-2!)
├─ fft_y = FFT(b)  → radix2FFT (2048 is power-of-2!)
├─ multiply: fft_x[i] * fft_y[i]  (pointwise multiplication)
└─ return IFFT(result)  → radix2FFT (inverse transform)
```

**Convolution Theorem:** `Convolution(a,b) = IFFT(FFT(a) * FFT(b))`

### Step 6: Post-processing
```go
result[i] *= invFactors[i]  // Final multiplication by e^(-πi*i²/860)
return result[:860]         // Trim back to original size
```

## Key Algorithms Explained

### Radix-2 FFT (Cooley-Tukey Algorithm)
**Used for power-of-2 sizes (like the 2048 in Bluestein's convolution)**

**Core Principle:** Divide-and-conquer approach
- Recursively split N-point DFT into two (N/2)-point DFTs
- Combine results using "butterfly" operations
- Complexity: O(N log N) instead of O(N²)

**Butterfly Operation:**
```go
t[idx] = ridx + w_n      // Even part
t[idx2] = ridx - w_n     // Odd part
```

**Twiddle Factors:** `W_N^k = e^(-2πi*k/N) = cos(2πk/N) - i*sin(2πk/N)`
- Complex exponentials that rotate values in the complex plane
- Enable the mathematical relationships for splitting DFTs

### Convolution Deep Dive
**Mathematical Definition:**
```
(f * g)[n] = Σ f[m] * g[n-m]
```

**Intuitive Process:**
1. Flip one signal
2. Slide it across the other signal
3. Multiply overlapping values
4. Sum the products
5. Move to next position and repeat

**Why Used in Bluestein:** The DFT can be mathematically rearranged into convolution form, allowing efficient computation using power-of-2 FFTs.

## Recursive Structure Analysis

### Total FFT Calls for One 860-Sample Chunk
1. **bluesteinFFT(860)** → Uses Bluestein algorithm
2. **Convolution step:**
   - `FFT(2048)` → radix2FFT (for array 'a')
   - `FFT(2048)` → radix2FFT (for array 'b')  
   - `IFFT(2048)` → radix2FFT (inverse)

**Result:** 4 total FFT operations (1 Bluestein + 3 power-of-2)

### Algorithm Efficiency
- **Arbitrary size (860):** Uses complex Bluestein algorithm
- **Power-of-2 sizes (2048):** Uses fast radix-2 algorithm
- **Strategy:** Transform hard problem into multiple easy problems

## Frequency Domain Clarification

### Common Confusion
**Question:** "Why IFFT when we want frequency domain data?"

**Answer:** The IFFT in convolution is NOT converting final results to time domain!

**Data Flow:**
1. **Goal:** Time domain → Frequency domain (for spectral analysis)
2. **Bluestein method:** Uses convolution to compute DFT efficiently  
3. **Convolution implementation:** Uses frequency domain math internally
4. **Final output:** IS frequency domain data (computed via convolution)

**Key Point:** Bluestein outputs frequency domain data - it just uses convolution as an internal computational method.

## Implementation Notes for Our Project

### Chunk Processing
- **Input:** 860 time-domain samples per chunk
- **Algorithm:** Bluestein FFT (due to non-power-of-2 size)
- **Output:** 860 frequency-domain complex values
- **Usage:** Extract magnitude spectrum for peak detection

### Performance Characteristics
- **Padding overhead:** 860 → 2048 samples (2.38x increase)
- **Multiple FFT calls:** 4 total operations per chunk
- **Trade-off:** Slightly slower than power-of-2, but maintains exact chunk size for optimal frequency resolution

### Next Steps
The frequency domain output will be used for:
1. Magnitude spectrum extraction
2. Spectral peak detection  
3. Constellation map generation
4. Audio fingerprint creation

## References and Further Reading
- Bluestein, L. (1970). A linear filtering approach to the computation of discrete Fourier transform
- Cooley, J.W.; Tukey, J.W. (1965). An algorithm for the machine calculation of complex Fourier series
- Understanding Digital Signal Processing by Richard G. Lyons
- The Scientist and Engineer's Guide to Digital Signal Processing by Steven W. Smith