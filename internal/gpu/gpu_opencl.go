//go:build !darwin && cgo

package gpu

/*
#cgo linux LDFLAGS: -lOpenCL
#cgo windows LDFLAGS: -lOpenCL

#ifdef __APPLE__
#include <OpenCL/cl.h>
#else
#include "clheaders/CL/cl.h"
#endif

#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// SHA-256 + Base32 prefix check OpenCL kernel
static const char* kernelSource =
"// SHA-256 round constants\n"
"__constant uint K[64] = {\n"
"    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,\n"
"    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,\n"
"    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,\n"
"    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,\n"
"    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,\n"
"    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,\n"
"    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,\n"
"    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,\n"
"    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,\n"
"    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,\n"
"    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,\n"
"    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,\n"
"    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,\n"
"    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,\n"
"    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,\n"
"    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2\n"
"};\n"
"\n"
"__constant char B32[32] = {\n"
"    'a','b','c','d','e','f','g','h','i','j','k','l','m',\n"
"    'n','o','p','q','r','s','t','u','v','w','x','y','z',\n"
"    '2','3','4','5','6','7'\n"
"};\n"
"\n"
"uint rotr_u(uint x, uint n) { return (x >> n) | (x << (32 - n)); }\n"
"uint ch_u(uint x, uint y, uint z) { return (x & y) ^ (~x & z); }\n"
"uint maj_u(uint x, uint y, uint z) { return (x & y) ^ (x & z) ^ (y & z); }\n"
"uint ep0_u(uint x) { return rotr_u(x, 2) ^ rotr_u(x, 13) ^ rotr_u(x, 22); }\n"
"uint ep1_u(uint x) { return rotr_u(x, 6) ^ rotr_u(x, 11) ^ rotr_u(x, 25); }\n"
"uint sig0_u(uint x) { return rotr_u(x, 7) ^ rotr_u(x, 18) ^ (x >> 3); }\n"
"uint sig1_u(uint x) { return rotr_u(x, 17) ^ rotr_u(x, 19) ^ (x >> 10); }\n"
"\n"
"__kernel void vanity_search(\n"
"    __global const uchar* dest_template,\n"
"    const ulong counter_base,\n"
"    const uint prefix_len,\n"
"    __global const char* prefix,\n"
"    __global int* match_found,\n"
"    __global ulong* match_counter\n"
") {\n"
"    ulong gid = get_global_id(0);\n"
"    ulong counter = counter_base + gid;\n"
"\n"
"    if (*match_found != 0) return;\n"
"\n"
"    uchar dest[448];\n"
"    for (uint i = 0; i < 391; i++) dest[i] = dest_template[i];\n"
"\n"
"    dest[0] = (uchar)(counter);\n"
"    dest[1] = (uchar)(counter >> 8);\n"
"    dest[2] = (uchar)(counter >> 16);\n"
"    dest[3] = (uchar)(counter >> 24);\n"
"    dest[4] = (uchar)(counter >> 32);\n"
"    dest[5] = (uchar)(counter >> 40);\n"
"    dest[6] = (uchar)(counter >> 48);\n"
"    dest[7] = (uchar)(counter >> 56);\n"
"\n"
"    dest[391] = 0x80;\n"
"    for (uint i = 392; i < 440; i++) dest[i] = 0;\n"
"    dest[440] = 0; dest[441] = 0; dest[442] = 0; dest[443] = 0;\n"
"    dest[444] = 0; dest[445] = 0; dest[446] = 0x0C; dest[447] = 0x38;\n"
"\n"
"    uint h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a;\n"
"    uint h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19;\n"
"\n"
"    for (uint block = 0; block < 7; block++) {\n"
"        uint w[64];\n"
"        uint base = block * 64;\n"
"        for (uint i = 0; i < 16; i++) {\n"
"            uint off = base + i * 4;\n"
"            w[i] = ((uint)dest[off] << 24) | ((uint)dest[off+1] << 16) |\n"
"                   ((uint)dest[off+2] << 8) | (uint)dest[off+3];\n"
"        }\n"
"        for (uint i = 16; i < 64; i++)\n"
"            w[i] = sig1_u(w[i-2]) + w[i-7] + sig0_u(w[i-15]) + w[i-16];\n"
"\n"
"        uint a = h0, b = h1, c = h2, d = h3;\n"
"        uint e = h4, f = h5, g = h6, h = h7;\n"
"        for (uint i = 0; i < 64; i++) {\n"
"            uint t1 = h + ep1_u(e) + ch_u(e, f, g) + K[i] + w[i];\n"
"            uint t2 = ep0_u(a) + maj_u(a, b, c);\n"
"            h = g; g = f; f = e; e = d + t1;\n"
"            d = c; c = b; b = a; a = t1 + t2;\n"
"        }\n"
"        h0 += a; h1 += b; h2 += c; h3 += d;\n"
"        h4 += e; h5 += f; h6 += g; h7 += h;\n"
"    }\n"
"\n"
"    uchar hash[32];\n"
"    hash[0]=(h0>>24); hash[1]=(h0>>16); hash[2]=(h0>>8); hash[3]=h0;\n"
"    hash[4]=(h1>>24); hash[5]=(h1>>16); hash[6]=(h1>>8); hash[7]=h1;\n"
"    hash[8]=(h2>>24); hash[9]=(h2>>16); hash[10]=(h2>>8); hash[11]=h2;\n"
"    hash[12]=(h3>>24); hash[13]=(h3>>16); hash[14]=(h3>>8); hash[15]=h3;\n"
"    hash[16]=(h4>>24); hash[17]=(h4>>16); hash[18]=(h4>>8); hash[19]=h4;\n"
"    hash[20]=(h5>>24); hash[21]=(h5>>16); hash[22]=(h5>>8); hash[23]=h5;\n"
"    hash[24]=(h6>>24); hash[25]=(h6>>16); hash[26]=(h6>>8); hash[27]=h6;\n"
"    hash[28]=(h7>>24); hash[29]=(h7>>16); hash[30]=(h7>>8); hash[31]=h7;\n"
"\n"
"    uint bit_offset = 0;\n"
"    int match = 1;\n"
"    for (uint i = 0; i < prefix_len; i++) {\n"
"        uint byte_idx = bit_offset / 8;\n"
"        uint bit_idx = bit_offset % 8;\n"
"        uint val;\n"
"        if (bit_idx <= 3)\n"
"            val = (hash[byte_idx] >> (3 - bit_idx)) & 0x1f;\n"
"        else\n"
"            val = ((hash[byte_idx] << (bit_idx - 3)) | (hash[byte_idx + 1] >> (11 - bit_idx))) & 0x1f;\n"
"        if (B32[val] != prefix[i]) { match = 0; break; }\n"
"        bit_offset += 5;\n"
"    }\n"
"\n"
"    if (match) {\n"
"        if (atomic_cmpxchg(match_found, 0, 1) == 0) {\n"
"            *match_counter = counter;\n"
"        }\n"
"    }\n"
"}\n";

// SHA3-256 (Keccak) + Base32 prefix check kernel for Tor v3 .onion addresses
static const char* torV3KernelSource =
"// Keccak-f[1600] round constants\n"
"__constant ulong RC[24] = {\n"
"    0x0000000000000001UL, 0x0000000000008082UL, 0x800000000000808AUL,\n"
"    0x8000000080008000UL, 0x000000000000808BUL, 0x0000000080000001UL,\n"
"    0x8000000080008081UL, 0x8000000000008009UL, 0x000000000000008AUL,\n"
"    0x0000000000000088UL, 0x0000000080008009UL, 0x000000008000000AUL,\n"
"    0x000000008000808BUL, 0x800000000000008BUL, 0x8000000000008089UL,\n"
"    0x8000000000008003UL, 0x8000000000008002UL, 0x8000000000000080UL,\n"
"    0x000000000000800AUL, 0x800000008000000AUL, 0x8000000080008081UL,\n"
"    0x8000000000008080UL, 0x0000000080000001UL, 0x8000000080008008UL\n"
"};\n"
"\n"
"// Keccak rho rotation offsets (indexed as [x + 5*y])\n"
"__constant uint RHO[25] = {\n"
"    0, 1, 62, 28, 27, 36, 44, 6, 55, 20,\n"
"    3, 10, 43, 25, 39, 41, 45, 15, 21, 8,\n"
"    18, 2, 61, 56, 14\n"
"};\n"
"\n"
"// Keccak pi permutation: PI[i] = destination index for source i\n"
"__constant int PI[25] = {\n"
"    0, 10, 20, 5, 15, 16, 1, 11, 21, 6,\n"
"    7, 17, 2, 12, 22, 23, 8, 18, 3, 13,\n"
"    14, 24, 9, 19, 4\n"
"};\n"
"\n"
"// Base32 alphabet (RFC 4648, lowercase)\n"
"__constant char B32T[32] = {\n"
"    'a','b','c','d','e','f','g','h','i','j','k','l','m',\n"
"    'n','o','p','q','r','s','t','u','v','w','x','y','z',\n"
"    '2','3','4','5','6','7'\n"
"};\n"
"\n"
"ulong rotl64(ulong x, uint n) {\n"
"    return (n == 0) ? x : ((x << n) | (x >> (64 - n)));\n"
"}\n"
"\n"
"__kernel void torv3_check(\n"
"    __global const uchar* pubkeys,\n"
"    const uint key_count,\n"
"    const uint prefix_len,\n"
"    __global const char* prefix,\n"
"    __global int* match_found,\n"
"    __global ulong* match_index\n"
") {\n"
"    uint gid = get_global_id(0);\n"
"    if (gid >= key_count) return;\n"
"    if (*match_found != 0) return;\n"
"\n"
"    __global const uchar* pk = pubkeys + gid * 32;\n"
"\n"
"    // --- SHA3-256(\".onion checksum\" || pubkey || 0x03) ---\n"
"    // Message is 48 bytes, fits in one SHA3-256 block (rate=136)\n"
"    ulong state[25];\n"
"    for (int i = 0; i < 25; i++) state[i] = 0;\n"
"\n"
"    // Absorb message + SHA3 padding directly as LE uint64 words\n"
"    // Word 0: bytes 0-7 = \".onion c\"\n"
"    state[0] = 0x63206e6f696e6f2eUL;\n"
"    // Word 1: bytes 8-15 = \"hecksum\" + pk[0]\n"
"    state[1] = 0x006d75736b636568UL | ((ulong)pk[0] << 56);\n"
"    // Word 2: pk[1..8]\n"
"    state[2] = (ulong)pk[1] | ((ulong)pk[2]<<8) | ((ulong)pk[3]<<16) | ((ulong)pk[4]<<24) |\n"
"              ((ulong)pk[5]<<32) | ((ulong)pk[6]<<40) | ((ulong)pk[7]<<48) | ((ulong)pk[8]<<56);\n"
"    // Word 3: pk[9..16]\n"
"    state[3] = (ulong)pk[9] | ((ulong)pk[10]<<8) | ((ulong)pk[11]<<16) | ((ulong)pk[12]<<24) |\n"
"              ((ulong)pk[13]<<32) | ((ulong)pk[14]<<40) | ((ulong)pk[15]<<48) | ((ulong)pk[16]<<56);\n"
"    // Word 4: pk[17..24]\n"
"    state[4] = (ulong)pk[17] | ((ulong)pk[18]<<8) | ((ulong)pk[19]<<16) | ((ulong)pk[20]<<24) |\n"
"              ((ulong)pk[21]<<32) | ((ulong)pk[22]<<40) | ((ulong)pk[23]<<48) | ((ulong)pk[24]<<56);\n"
"    // Word 5: pk[25..31] + version 0x03\n"
"    state[5] = (ulong)pk[25] | ((ulong)pk[26]<<8) | ((ulong)pk[27]<<16) | ((ulong)pk[28]<<24) |\n"
"              ((ulong)pk[29]<<32) | ((ulong)pk[30]<<40) | ((ulong)pk[31]<<48) | (0x03UL<<56);\n"
"    // Word 6: SHA3 domain separator 0x06 at byte 48\n"
"    state[6] = 0x0000000000000006UL;\n"
"    // Words 7-15: zero (already initialized)\n"
"    // Word 16: padding end 0x80 at byte 135\n"
"    state[16] = 0x8000000000000000UL;\n"
"\n"
"    // Keccak-f[1600]: 24 rounds\n"
"    for (int round = 0; round < 24; round++) {\n"
"        // Theta\n"
"        ulong C[5];\n"
"        for (int x = 0; x < 5; x++)\n"
"            C[x] = state[x] ^ state[x+5] ^ state[x+10] ^ state[x+15] ^ state[x+20];\n"
"        for (int x = 0; x < 5; x++) {\n"
"            ulong D = C[(x+4)%5] ^ rotl64(C[(x+1)%5], 1);\n"
"            for (int y = 0; y < 5; y++)\n"
"                state[x + 5*y] ^= D;\n"
"        }\n"
"        // Rho + Pi\n"
"        ulong B[25];\n"
"        for (int i = 0; i < 25; i++)\n"
"            B[PI[i]] = rotl64(state[i], RHO[i]);\n"
"        // Chi\n"
"        for (int y = 0; y < 5; y++) {\n"
"            for (int x = 0; x < 5; x++)\n"
"                state[x + 5*y] = B[x + 5*y] ^ (~B[((x+1)%5) + 5*y] & B[((x+2)%5) + 5*y]);\n"
"        }\n"
"        // Iota\n"
"        state[0] ^= RC[round];\n"
"    }\n"
"\n"
"    // Extract checksum: first 2 bytes of SHA3-256 output (LE from state[0])\n"
"    uchar cksum0 = (uchar)(state[0]);\n"
"    uchar cksum1 = (uchar)(state[0] >> 8);\n"
"\n"
"    // Build 35-byte payload: pubkey(32) | checksum(2) | version(1)\n"
"    uchar payload[35];\n"
"    for (int i = 0; i < 32; i++) payload[i] = pk[i];\n"
"    payload[32] = cksum0;\n"
"    payload[33] = cksum1;\n"
"    payload[34] = 0x03;\n"
"\n"
"    // Base32 encode and compare prefix\n"
"    uint bit_offset = 0;\n"
"    int match = 1;\n"
"    for (uint i = 0; i < prefix_len; i++) {\n"
"        uint byte_idx = bit_offset / 8;\n"
"        uint bit_idx = bit_offset % 8;\n"
"        uint val;\n"
"        if (bit_idx <= 3)\n"
"            val = (payload[byte_idx] >> (3 - bit_idx)) & 0x1f;\n"
"        else\n"
"            val = ((payload[byte_idx] << (bit_idx - 3)) | (payload[byte_idx + 1] >> (11 - bit_idx))) & 0x1f;\n"
"        if (B32T[val] != prefix[i]) { match = 0; break; }\n"
"        bit_offset += 5;\n"
"    }\n"
"\n"
"    if (match) {\n"
"        if (atomic_cmpxchg(match_found, 0, 1) == 0) {\n"
"            *match_index = (ulong)gid;\n"
"        }\n"
"    }\n"
"}\n";

typedef struct {
    cl_context context;
    cl_command_queue queue;
    cl_kernel kernel;
    cl_program program;
    cl_mem destBuf;
    cl_mem prefixBuf;
    cl_mem matchFoundBuf;
    cl_mem matchCounterBuf;
    cl_ulong batchSize;
    int prefixLen;
} OpenCLWorker;

static cl_device_id* g_devices = NULL;
static int g_deviceCount = 0;
static int g_initialized = 0;

static void ensureInit(void) {
    if (g_initialized) return;
    g_initialized = 1;

    cl_uint numPlatforms = 0;
    cl_int err = clGetPlatformIDs(0, NULL, &numPlatforms);
    if (err != CL_SUCCESS || numPlatforms == 0) {
        fprintf(stderr, "[gpu] no OpenCL platforms found (err=%d): no OpenCL runtime/driver (ICD) is installed\n", err);
#ifdef __linux__
        fprintf(stderr, "[gpu] AMD on Linux: install an OpenCL ICD, e.g. 'sudo apt install mesa-opencl-icd' (Debian/Ubuntu) or ROCm, then verify with 'clinfo'\n");
        fprintf(stderr, "[gpu] NVIDIA on Linux: the proprietary driver provides OpenCL (package 'nvidia-opencl-icd' on Debian)\n");
#endif
        return;
    }

    cl_platform_id* platforms = (cl_platform_id*)malloc(sizeof(cl_platform_id) * numPlatforms);
    clGetPlatformIDs(numPlatforms, platforms, NULL);

    // Count all GPU devices across platforms
    int total = 0;
    for (cl_uint p = 0; p < numPlatforms; p++) {
        char pname[256] = "?", pver[256] = "?";
        clGetPlatformInfo(platforms[p], CL_PLATFORM_NAME, sizeof(pname), pname, NULL);
        clGetPlatformInfo(platforms[p], CL_PLATFORM_VERSION, sizeof(pver), pver, NULL);
        cl_uint nd = 0;
        err = clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, 0, NULL, &nd);
        if (err != CL_SUCCESS && err != CL_DEVICE_NOT_FOUND)
            fprintf(stderr, "[gpu] platform %u \"%s\" (%s): clGetDeviceIDs error %d\n", p, pname, pver, err);
        else
            fprintf(stderr, "[gpu] platform %u \"%s\" (%s): %u GPU device(s)\n", p, pname, pver, nd);
        total += nd;
    }
    if (total == 0) {
        fprintf(stderr, "[gpu] OpenCL is installed but no platform exposes a GPU device; GPU acceleration disabled\n");
        free(platforms);
        return;
    }

    g_devices = (cl_device_id*)malloc(sizeof(cl_device_id) * total);
    int idx = 0;
    for (cl_uint p = 0; p < numPlatforms; p++) {
        cl_uint nd = 0;
        clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, 0, NULL, &nd);
        if (nd > 0) {
            clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, nd, g_devices + idx, NULL);
            idx += nd;
        }
    }
    g_deviceCount = idx;
    for (int i = 0; i < g_deviceCount; i++) {
        char dname[256] = "?", dvendor[256] = "?", dver[256] = "?";
        clGetDeviceInfo(g_devices[i], CL_DEVICE_NAME, sizeof(dname), dname, NULL);
        clGetDeviceInfo(g_devices[i], CL_DEVICE_VENDOR, sizeof(dvendor), dvendor, NULL);
        clGetDeviceInfo(g_devices[i], CL_DEVICE_VERSION, sizeof(dver), dver, NULL);
        fprintf(stderr, "[gpu] device %d: %s (%s, %s)\n", i, dname, dvendor, dver);
    }
    free(platforms);
}

int oclAvailable(void) {
    ensureInit();
    return g_deviceCount > 0 ? 1 : 0;
}

int oclDeviceCount(void) {
    ensureInit();
    return g_deviceCount;
}

char* oclDeviceName(int index) {
    ensureInit();
    if (index < 0 || index >= g_deviceCount) return strdup("Unknown");
    char name[256];
    clGetDeviceInfo(g_devices[index], CL_DEVICE_NAME, sizeof(name), name, NULL);
    return strdup(name);
}

char* oclDeviceVendor(int index) {
    ensureInit();
    if (index < 0 || index >= g_deviceCount) return strdup("Unknown");
    char vendor[256];
    clGetDeviceInfo(g_devices[index], CL_DEVICE_VENDOR, sizeof(vendor), vendor, NULL);
    return strdup(vendor);
}

// Compiles and runs a tiny kernel exercising the features the real kernels
// need (global int32 atomic_cmpxchg, 64-bit integers). Returns 1 on success.
// Run from a disposable subprocess: broken driver compilers (e.g. Mesa r600)
// abort() instead of returning a build error, killing the calling process.
int oclProbeDevice(int index) {
    ensureInit();
    if (index < 0 || index >= g_deviceCount) return 0;

    cl_device_id dev = g_devices[index];
    cl_int err;
    const char* src =
        "__kernel void probe(__global int* flag, __global ulong* out) {\n"
        "    ulong gid = get_global_id(0);\n"
        "    if (atomic_cmpxchg(flag, 0, 1) == 0) { out[0] = gid + 1; }\n"
        "}\n";

    cl_context ctx = clCreateContext(NULL, 1, &dev, NULL, NULL, &err);
    if (err != CL_SUCCESS) return 0;
    cl_command_queue q = clCreateCommandQueue(ctx, dev, 0, &err);
    if (err != CL_SUCCESS) { clReleaseContext(ctx); return 0; }

    int ok = 0;
    size_t len = strlen(src);
    cl_program prog = clCreateProgramWithSource(ctx, 1, &src, &len, &err);
    if (err == CL_SUCCESS) {
        err = clBuildProgram(prog, 1, &dev, NULL, NULL, NULL);
        if (err != CL_SUCCESS) {
            char log[2048];
            clGetProgramBuildInfo(prog, dev, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
            fprintf(stderr, "[gpu] probe build failed on device %d: %s\n", index, log);
        } else {
            cl_kernel k = clCreateKernel(prog, "probe", &err);
            if (err == CL_SUCCESS) {
                int zero = 0;
                cl_ulong outv = 0;
                cl_mem flag = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                             sizeof(int), &zero, &err);
                cl_mem out = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                            sizeof(cl_ulong), &outv, &err);
                clSetKernelArg(k, 0, sizeof(cl_mem), &flag);
                clSetKernelArg(k, 1, sizeof(cl_mem), &out);
                size_t gs = 64;
                if (clEnqueueNDRangeKernel(q, k, 1, NULL, &gs, NULL, 0, NULL, NULL) == CL_SUCCESS &&
                    clFinish(q) == CL_SUCCESS) {
                    cl_ulong res = 0;
                    if (clEnqueueReadBuffer(q, out, CL_TRUE, 0, sizeof(cl_ulong), &res,
                                            0, NULL, NULL) == CL_SUCCESS && res != 0)
                        ok = 1;
                }
                clReleaseMemObject(out);
                clReleaseMemObject(flag);
                clReleaseKernel(k);
            }
        }
        clReleaseProgram(prog);
    }
    clReleaseCommandQueue(q);
    clReleaseContext(ctx);
    return ok;
}

void* oclNewWorker(int deviceIndex, const unsigned char* destTemplate,
                   const char* prefix, int prefixLen, unsigned long batchSize) {
    ensureInit();
    if (deviceIndex < 0 || deviceIndex >= g_deviceCount) return NULL;

    cl_device_id dev = g_devices[deviceIndex];
    cl_int err;

    cl_context ctx = clCreateContext(NULL, 1, &dev, NULL, NULL, &err);
    if (err != CL_SUCCESS) return NULL;

    cl_command_queue queue = clCreateCommandQueue(ctx, dev, 0, &err);
    if (err != CL_SUCCESS) { clReleaseContext(ctx); return NULL; }

    const char* src = kernelSource;
    size_t srcLen = strlen(kernelSource);
    cl_program prog = clCreateProgramWithSource(ctx, 1, &src, &srcLen, &err);
    if (err != CL_SUCCESS) { clReleaseCommandQueue(queue); clReleaseContext(ctx); return NULL; }

    err = clBuildProgram(prog, 1, &dev, NULL, NULL, NULL);
    if (err != CL_SUCCESS) {
        char log[4096];
        clGetProgramBuildInfo(prog, dev, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
        fprintf(stderr, "OpenCL build error: %s\n", log);
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    cl_kernel kern = clCreateKernel(prog, "vanity_search", &err);
    if (err != CL_SUCCESS) {
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    cl_mem destBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                                    391, (void*)destTemplate, &err);
    cl_mem prefixBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                                      prefixLen, (void*)prefix, &err);

    int zero = 0;
    cl_mem matchFoundBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                          sizeof(int), &zero, &err);
    cl_ulong zeroCounter = 0;
    cl_mem matchCounterBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                            sizeof(cl_ulong), &zeroCounter, &err);

    // Set static kernel args
    clSetKernelArg(kern, 0, sizeof(cl_mem), &destBuf);
    // arg 1 (counter_base) set per batch
    cl_uint pl = (cl_uint)prefixLen;
    clSetKernelArg(kern, 2, sizeof(cl_uint), &pl);
    clSetKernelArg(kern, 3, sizeof(cl_mem), &prefixBuf);
    clSetKernelArg(kern, 4, sizeof(cl_mem), &matchFoundBuf);
    clSetKernelArg(kern, 5, sizeof(cl_mem), &matchCounterBuf);

    OpenCLWorker* w = (OpenCLWorker*)calloc(1, sizeof(OpenCLWorker));
    w->context = ctx;
    w->queue = queue;
    w->kernel = kern;
    w->program = prog;
    w->destBuf = destBuf;
    w->prefixBuf = prefixBuf;
    w->matchFoundBuf = matchFoundBuf;
    w->matchCounterBuf = matchCounterBuf;
    w->batchSize = (cl_ulong)batchSize;
    w->prefixLen = prefixLen;
    return w;
}

unsigned long oclRunBatch(void* handle, unsigned long counterStart,
                          int* matchFound, unsigned long* matchCounter) {
    OpenCLWorker* w = (OpenCLWorker*)handle;
    if (!w) return 0;

    // Reset match_found
    int zero = 0;
    clEnqueueWriteBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0, sizeof(int), &zero, 0, NULL, NULL);

    // Set counter_base arg
    cl_ulong cb = (cl_ulong)counterStart;
    clSetKernelArg(w->kernel, 1, sizeof(cl_ulong), &cb);

    // Dispatch
    size_t globalSize = (size_t)w->batchSize;
    cl_int err = clEnqueueNDRangeKernel(w->queue, w->kernel, 1, NULL, &globalSize, NULL, 0, NULL, NULL);
    if (err != CL_SUCCESS) return 0;

    clFinish(w->queue);

    // Read results
    int mf = 0;
    cl_ulong mc = 0;
    clEnqueueReadBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0, sizeof(int), &mf, 0, NULL, NULL);
    clEnqueueReadBuffer(w->queue, w->matchCounterBuf, CL_TRUE, 0, sizeof(cl_ulong), &mc, 0, NULL, NULL);

    *matchFound = mf;
    *matchCounter = (unsigned long)mc;
    return (unsigned long)w->batchSize;
}

void oclFreeWorker(void* handle) {
    OpenCLWorker* w = (OpenCLWorker*)handle;
    if (!w) return;
    clReleaseMemObject(w->matchCounterBuf);
    clReleaseMemObject(w->matchFoundBuf);
    clReleaseMemObject(w->prefixBuf);
    clReleaseMemObject(w->destBuf);
    clReleaseKernel(w->kernel);
    clReleaseProgram(w->program);
    clReleaseCommandQueue(w->queue);
    clReleaseContext(w->context);
    free(w);
}

// ---- Tor v3 worker (SHA3-256 + base32 prefix check) ----

typedef struct {
    cl_context context;
    cl_command_queue queue;
    cl_kernel kernel;
    cl_program program;
    cl_mem pubkeyBuf;
    cl_mem prefixBuf;
    cl_mem matchFoundBuf;
    cl_mem matchIndexBuf;
    cl_ulong batchSize;
    int prefixLen;
} OpenCLTorV3Worker;

void* oclNewTorV3Worker(int deviceIndex, const char* prefix, int prefixLen,
                        unsigned long batchSize) {
    ensureInit();
    if (deviceIndex < 0 || deviceIndex >= g_deviceCount) return NULL;

    cl_device_id dev = g_devices[deviceIndex];
    cl_int err;

    cl_context ctx = clCreateContext(NULL, 1, &dev, NULL, NULL, &err);
    if (err != CL_SUCCESS) return NULL;

    cl_command_queue queue = clCreateCommandQueue(ctx, dev, 0, &err);
    if (err != CL_SUCCESS) { clReleaseContext(ctx); return NULL; }

    const char* src = torV3KernelSource;
    size_t srcLen = strlen(torV3KernelSource);
    cl_program prog = clCreateProgramWithSource(ctx, 1, &src, &srcLen, &err);
    if (err != CL_SUCCESS) { clReleaseCommandQueue(queue); clReleaseContext(ctx); return NULL; }

    err = clBuildProgram(prog, 1, &dev, NULL, NULL, NULL);
    if (err != CL_SUCCESS) {
        char log[4096];
        clGetProgramBuildInfo(prog, dev, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
        fprintf(stderr, "OpenCL build error (TorV3): %s\n", log);
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    cl_kernel kern = clCreateKernel(prog, "torv3_check", &err);
    if (err != CL_SUCCESS) {
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    // Allocate pubkey buffer (batchSize * 32 bytes)
    cl_mem pubkeyBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY,
                                       batchSize * 32, NULL, &err);
    cl_mem prefixBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                                       prefixLen, (void*)prefix, &err);

    int zero = 0;
    cl_mem matchFoundBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                           sizeof(int), &zero, &err);
    cl_ulong zeroIdx = 0;
    cl_mem matchIndexBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                           sizeof(cl_ulong), &zeroIdx, &err);

    // Set static kernel args
    clSetKernelArg(kern, 0, sizeof(cl_mem), &pubkeyBuf);
    // arg 1 (key_count) set per batch
    cl_uint pl = (cl_uint)prefixLen;
    clSetKernelArg(kern, 2, sizeof(cl_uint), &pl);
    clSetKernelArg(kern, 3, sizeof(cl_mem), &prefixBuf);
    clSetKernelArg(kern, 4, sizeof(cl_mem), &matchFoundBuf);
    clSetKernelArg(kern, 5, sizeof(cl_mem), &matchIndexBuf);

    OpenCLTorV3Worker* w = (OpenCLTorV3Worker*)calloc(1, sizeof(OpenCLTorV3Worker));
    w->context = ctx;
    w->queue = queue;
    w->kernel = kern;
    w->program = prog;
    w->pubkeyBuf = pubkeyBuf;
    w->prefixBuf = prefixBuf;
    w->matchFoundBuf = matchFoundBuf;
    w->matchIndexBuf = matchIndexBuf;
    w->batchSize = (cl_ulong)batchSize;
    w->prefixLen = prefixLen;
    return w;
}

unsigned long oclRunTorV3Batch(void* handle, const unsigned char* pubkeys,
                                unsigned long keyCount,
                                int* matchFound, unsigned long* matchIndex) {
    OpenCLTorV3Worker* w = (OpenCLTorV3Worker*)handle;
    if (!w) return 0;

    // Upload pubkeys
    clEnqueueWriteBuffer(w->queue, w->pubkeyBuf, CL_TRUE, 0,
                          keyCount * 32, pubkeys, 0, NULL, NULL);

    // Reset match_found
    int zero = 0;
    clEnqueueWriteBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0,
                          sizeof(int), &zero, 0, NULL, NULL);

    // Set key_count arg
    cl_uint kc = (cl_uint)keyCount;
    clSetKernelArg(w->kernel, 1, sizeof(cl_uint), &kc);

    // Dispatch
    size_t globalSize = (size_t)keyCount;
    cl_int err = clEnqueueNDRangeKernel(w->queue, w->kernel, 1, NULL,
                                         &globalSize, NULL, 0, NULL, NULL);
    if (err != CL_SUCCESS) return 0;

    clFinish(w->queue);

    // Read results
    int mf = 0;
    cl_ulong mi = 0;
    clEnqueueReadBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0,
                         sizeof(int), &mf, 0, NULL, NULL);
    clEnqueueReadBuffer(w->queue, w->matchIndexBuf, CL_TRUE, 0,
                         sizeof(cl_ulong), &mi, 0, NULL, NULL);

    *matchFound = mf;
    *matchIndex = (unsigned long)mi;
    return (unsigned long)keyCount;
}

void oclFreeTorV3Worker(void* handle) {
    OpenCLTorV3Worker* w = (OpenCLTorV3Worker*)handle;
    if (!w) return;
    clReleaseMemObject(w->matchIndexBuf);
    clReleaseMemObject(w->matchFoundBuf);
    clReleaseMemObject(w->prefixBuf);
    clReleaseMemObject(w->pubkeyBuf);
    clReleaseKernel(w->kernel);
    clReleaseProgram(w->program);
    clReleaseCommandQueue(w->queue);
    clReleaseContext(w->context);
    free(w);
}
*/
import "C"
import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
	"unsafe"
)

// Available returns true if at least one OpenCL GPU device is detected.
func Available() bool {
	return C.oclAvailable() != 0
}

// ListDevices enumerates OpenCL GPU devices and filters out devices whose
// driver cannot compile/run our kernels. Each candidate is probed in a
// subprocess because broken driver compilers (e.g. Mesa r600 on old AMD APUs)
// abort() instead of returning an error, which would kill the app.
func ListDevices() ([]Device, error) {
	count := int(C.oclDeviceCount())
	if count == 0 {
		return nil, nil
	}

	exe, exeErr := os.Executable()

	var devices []Device
	for i := 0; i < count; i++ {
		cName := C.oclDeviceName(C.int(i))
		cVendor := C.oclDeviceVendor(C.int(i))
		d := Device{
			Name:    C.GoString(cName),
			Vendor:  C.GoString(cVendor),
			Backend: "OpenCL",
			Index:   i,
		}
		C.free(unsafe.Pointer(cName))
		C.free(unsafe.Pointer(cVendor))

		if exeErr == nil && !probeDeviceSubprocess(exe, i) {
			fmt.Fprintf(os.Stderr, "[gpu] device %d (%s) failed compute probe; excluding\n", i, d.Name)
			continue
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// probeDeviceSubprocess re-executes this binary in probe mode against one
// device. Any failure — nonzero exit, driver abort(), or hang past the
// timeout — marks the device unusable.
func probeDeviceSubprocess(exe string, index int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(), ProbeEnv+"="+strconv.Itoa(index))
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

// ProbeMain is the subprocess entry point for probe mode (see ProbeEnv).
func ProbeMain(arg string) int {
	idx, err := strconv.Atoi(arg)
	if err != nil {
		return 2
	}
	if C.oclProbeDevice(C.int(idx)) != 0 {
		return 0
	}
	return 1
}

// NewWorker creates a GPU worker using OpenCL compute.
func NewWorker(cfg WorkerConfig) (*Worker, error) {
	if !Available() {
		return nil, fmt.Errorf("no OpenCL GPU available")
	}

	cPrefix := C.CString(cfg.Prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	handle := C.oclNewWorker(
		C.int(cfg.DeviceIndex),
		(*C.uchar)(unsafe.Pointer(&cfg.DestTemplate[0])),
		cPrefix,
		C.int(len(cfg.Prefix)),
		C.ulong(cfg.BatchSize),
	)
	if handle == nil {
		return nil, fmt.Errorf("failed to create OpenCL compute pipeline")
	}

	return &Worker{
		impl: &openclWorker{handle: handle},
	}, nil
}

type openclWorker struct {
	handle unsafe.Pointer
}

func (w *openclWorker) runBatch(counterStart uint64) (BatchResult, error) {
	var matchFound C.int
	var matchCounter C.ulong

	checked := C.oclRunBatch(w.handle, C.ulong(counterStart), &matchFound, &matchCounter)
	if checked == 0 {
		return BatchResult{}, fmt.Errorf("OpenCL kernel execution failed")
	}

	return BatchResult{
		Found:        matchFound != 0,
		MatchCounter: uint64(matchCounter),
		Checked:      uint64(checked),
	}, nil
}

func (w *openclWorker) close() {
	if w.handle != nil {
		C.oclFreeWorker(w.handle)
		w.handle = nil
	}
}

// NewTorV3Worker creates a GPU worker for Tor v3 vanity checking using OpenCL.
func NewTorV3Worker(cfg TorV3WorkerConfig) (*TorV3Worker, error) {
	if !Available() {
		return nil, fmt.Errorf("no OpenCL GPU available")
	}

	cPrefix := C.CString(cfg.Prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	handle := C.oclNewTorV3Worker(
		C.int(cfg.DeviceIndex),
		cPrefix,
		C.int(len(cfg.Prefix)),
		C.ulong(cfg.BatchSize),
	)
	if handle == nil {
		return nil, fmt.Errorf("failed to create OpenCL Tor v3 compute pipeline")
	}

	return &TorV3Worker{
		impl: &openclTorV3Worker{handle: handle},
	}, nil
}

type openclTorV3Worker struct {
	handle unsafe.Pointer
}

func (w *openclTorV3Worker) runBatch(pubkeys []byte, keyCount uint64) (BatchResult, error) {
	var matchFound C.int
	var matchIndex C.ulong

	checked := C.oclRunTorV3Batch(w.handle,
		(*C.uchar)(unsafe.Pointer(&pubkeys[0])),
		C.ulong(keyCount),
		&matchFound, &matchIndex)
	if checked == 0 {
		return BatchResult{}, fmt.Errorf("OpenCL Tor v3 kernel execution failed")
	}

	return BatchResult{
		Found:        matchFound != 0,
		MatchCounter: uint64(matchIndex),
		Checked:      uint64(checked),
	}, nil
}

func (w *openclTorV3Worker) close() {
	if w.handle != nil {
		C.oclFreeTorV3Worker(w.handle)
		w.handle = nil
	}
}
