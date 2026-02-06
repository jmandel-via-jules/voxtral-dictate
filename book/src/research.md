# Mistral Voxtral: Complete Speech-to-Text Report

*Compiled February 6, 2026*

## The Model Family

Mistral's speech-to-text offering is called **Voxtral**. As of this week (Transcribe 2 launched Feb 4, 2026), there are **five models**:

| Model | Params | Purpose | License | Released |
|---|---|---|---|---|
| **Voxtral Mini 3B** (2507) | ~5B (3B LLM + encoder) | Offline transcription + audio understanding | Apache 2.0 | Jul 2025 |
| **Voxtral Small 24B** (2507) | ~24B | Premium offline transcription + understanding | Apache 2.0 | Jul 2025 |
| **Voxtral Mini Transcribe V2** (2602) | ~4B (API-only) | Best batch transcription, diarization | Proprietary (API) | Feb 4, 2026 |
| **Voxtral Mini 4B Realtime** (2602) | ~4B (3.4B LLM + 0.6B encoder) | **Streaming/realtime transcription** | Apache 2.0 | Feb 4, 2026 |
| Voxtral Mini Transcribe Realtime (API) | Same as above | API-hosted version of Realtime | API | Feb 4, 2026 |

The **Realtime 4B** model is most relevant to your dictation use case — purpose-built for live transcription with configurable latency from sub-200ms to 2.4s, open-weight Apache 2.0.

Research paper: [arXiv:2507.13264](https://arxiv.org/abs/2507.13264)

---

## Architecture (Realtime 4B)

**Audio Encoder (~0.6B params):**
- Whisper-style causal encoder trained from scratch
- 1280 dims, 32 layers, 32 heads, SwiGLU FFN
- **Causal attention** (key for streaming — processes audio as it arrives)
- Sliding window attention: 750
- 16kHz sampling, 128 mel bins, frame_rate 12.5 Hz, downsample factor 4

**Language Model (~3.4B params):**
- Based on Ministral-3-3B-Base-2512
- 3072 dims, 26 layers, 32 heads (8 KV heads)
- Sliding window attention: 8192
- Max model length: 131,072 tokens (~3+ hours audio)
- 1 text token = 80ms of audio

---

## Hardware Requirements

### Voxtral Mini 3B (offline transcription)
| Precision | VRAM/RAM | Notes |
|---|---|---|
| BF16/FP16 | **~9.5 GB VRAM** | Official figure; fits RTX 3060 12GB |
| FP8 (RedHatAI) | **~5 GB VRAM** | `RedHatAI/Voxtral-Mini-3B-2507-FP8-dynamic` |
| GGUF Q8_0 | **~4.3 GB** | Near-lossless |
| GGUF Q4_K_M | **~2.5 GB** | Recommended tradeoff |
| GGUF Q2_K | **~1.7 GB** | Surprisingly usable |

### Voxtral Mini 4B Realtime (streaming)
| Precision | VRAM | Notes |
|---|---|---|
| BF16 | **≥16 GB VRAM** | Official minimum; RTX 4060 Ti 16GB, RTX 3090, RTX 4080+ |
| No quants yet | — | Novel architecture, only vLLM currently |

### Voxtral Small 24B (premium quality)
| Precision | VRAM | Notes |
|---|---|---|
| BF16/FP16 | **~55 GB** | Needs 2× A100 or similar |
| FP8 | **~27.5 GB** | Fits single A100/H100 |
| GGUF Q4_K_M | **~14.3 GB** | Fits RTX 4090 (24GB) |
| 4-bit mixed | **~13-15 GB** | Community quants |

---

## All Quantized / Optimized Versions

### GGUF (llama.cpp) — Voxtral Mini 3B

| Source | Quants Available | Downloads |
|---|---|---|
| **bartowski/mistralai_Voxtral-Mini-3B-2507-GGUF** | BF16 (8.0GB) → IQ2_M (1.56GB), full range with imatrix | 2,067/mo |
| **ggml-org/Voxtral-Mini-3B-2507-GGUF** | Q4_K_M (2.47GB) | 330/mo |
| **mradermacher/Voxtral-Mini-3B-2507-i1-GGUF** | imatrix quants | 243/mo |

Key quant file sizes:
| Quant | Size | Quality |
|---|---|---|
| BF16 | 8.04 GB | Full precision |
| Q8_0 | 4.27 GB | Near-lossless |
| Q6_K_L | 3.50 GB | Excellent |
| **Q4_K_M** | **2.47 GB** | **Best tradeoff** |
| IQ4_XS | 2.27 GB | Good, small |
| Q3_K_M | 2.06 GB | Usable |
| Q2_K | 1.66 GB | Surprisingly OK |

### GGUF — Voxtral Small 24B
| Source | Downloads |
|---|---|
| **bartowski/mistralai_Voxtral-Small-24B-2507-GGUF** | 1,510/mo |
| **mradermacher/Voxtral-Small-24B-2507-i1-GGUF** | 166/mo |

### FP8 (vLLM)
| Model | Source | Savings |
|---|---|---|
| Mini 3B | `RedHatAI/Voxtral-Mini-3B-2507-FP8-dynamic` | ~50% VRAM reduction |
| Small 24B | `RedHatAI/Voxtral-Small-24B-2507-FP8-dynamic` | ~50% VRAM reduction |

### bitsandbytes (4-bit/8-bit mixed)
| Model | Source |
|---|---|
| Mini 3B 4-bit | `mzbac/voxtral-mini-3b-4bit-mixed` (0.9B effective) |
| Small 24B 4-bit | `VincentGOURBIN/voxtral-small-4bit-mixed` |
| Small 24B 8-bit | `VincentGOURBIN/voxtral-small-8bit` (26.5GB) |

### Other Formats
| Format | Model | Source |
|---|---|---|
| ONNX (browser/WebGPU) | Mini 3B | `onnx-community/Voxtral-Mini-3B-2507-ONNX` |
| MLX (Apple Silicon) | Mini 3B | `mlx-community/Voxtral-Mini-3B-2507-bf16` |

### Voxtral Realtime 4B
- **No quantized versions exist yet** — novel architecture
- Only supported in vLLM currently
- HF model card: *"We very much welcome community contributions to add the architecture to Transformers and Llama.cpp"*

---

## All Ways to Access Voxtral

### 1. Mistral API — Realtime Streaming (Easiest, best latency)
- **Protocol:** WebSocket via `client.audio.realtime.transcribe_stream()`
- **Model:** `voxtral-mini-transcribe-realtime-2602`
- **Pricing:** **$0.006/min** ($0.36/hr)
- **Latency:** Sub-200ms configurable
- **SDK:** `pip install mistralai[realtime]`
- **Pros:** Zero setup, lowest latency, no hardware needed
- **Cons:** Requires internet, costs money, privacy implications

### 2. Mistral API — Batch Transcription
- **Endpoint:** `POST /v1/audio/transcriptions`
- **Model:** `voxtral-mini-latest` (points to `voxtral-mini-2602`)
- **Pricing:** **$0.003/min** ($0.18/hr); batch API gets 50% discount
- **Features:** Diarization, word timestamps, context biasing, 13 languages, up to 3hr audio
- **Not suitable for live dictation** (batch only)

### 3. Local vLLM — Voxtral Realtime 4B (Best local streaming)
- **Hardware:** GPU with ≥16 GB VRAM
- **Streaming:** ✅ True streaming via vLLM Realtime API (WebSocket `/v1/realtime`)
- **Setup:**
  ```bash
  uv pip install -U vllm --torch-backend=auto --extra-index-url https://wheels.vllm.ai/nightly
  uv pip install soxr librosa soundfile
  VLLM_DISABLE_COMPILE_CACHE=1 vllm serve mistralai/Voxtral-Mini-4B-Realtime-2602 \
    --compilation_config '{"cudagraph_mode": "PIECEWISE"}'
  ```
- **Pros:** Fully local/private, true streaming, production-grade
- **Cons:** ≥16 GB VRAM required, vLLM nightly only

### 4. Local voxtral.c — Pure C Inference (Novel option)
- **Author:** antirez (creator of Redis)
- **Repo:** `github.com/antirez/voxtral.c`
- **Hardware:** ~8.9 GB for model; supports Metal (Apple), OpenBLAS (CPU)
- **Streaming:** ✅ Native C streaming API (`vox_stream_t`) — feed audio incrementally, get tokens back
- **Also supports:** stdin piping from ffmpeg
- **Pros:** Zero dependencies, lean, streaming native
- **Cons:** No GPU acceleration on NVIDIA yet (Metal + OpenBLAS only), early-stage

### 5. Local llama.cpp — Voxtral Mini 3B GGUF (Lowest hardware)
- **Hardware:** As little as 1.5-2.5 GB RAM/VRAM
- **Audio support:** ✅ Merged (PR #14862) via `llama-mtmd-cli` and `llama-server` multimodal support
- **Streaming:** ⚠️ Token streaming via `/v1/chat/completions` with audio input — not true audio streaming
- **No `/v1/audio/transcriptions` endpoint** (requested in issue #15291, closed as stale)
- **Setup:**
  ```bash
  llama-server -hf bartowski/mistralai_Voxtral-Mini-3B-2507-GGUF \
    -hff Voxtral-Mini-3B-2507-Q4_K_M.gguf
  ```
- **Pros:** Runs on CPU, tiny footprint, mature ecosystem
- **Cons:** Not true audio streaming; record chunk → send → get text. Higher latency.

### 6. Local vLLM — Voxtral Mini 3B (Offline, good quality)
- **Hardware:** GPU with ≥10 GB VRAM
- **Streaming:** ⚠️ Token streaming only, not live audio streaming
- **Setup:**
  ```bash
  vllm serve mistralai/Voxtral-Mini-3B-2507 \
    --tokenizer_mode mistral --config_format mistral --load_format mistral
  ```

### 7. Local HuggingFace Transformers (Python-native)
- **Hardware:** GPU with ≥10 GB VRAM (bf16); <5 GB with quantization
- **Framework:** `transformers >= 4.54.0` via `VoxtralForConditionalGeneration`
- **Streaming:** ⚠️ Token streaming only
- **Supports:** Multi-audio, multi-turn, batched inference

### 8. DeepInfra — Cloud API
- **Model:** `mistralai/Voxtral-Mini-3B-2507`
- **Pricing:** **$0.001/min** (cheapest cloud option)
- **Streaming:** ❌ Batch upload only

### 9. AWS Bedrock
- Voxtral Mini/Small 1.0 available but as **chat models** (not dedicated STT)
- No streaming transcription endpoint

### 10. AWS SageMaker — Self-Deploy
- Deploy via vLLM + custom Docker container
- Full control over streaming
- Instance types: `ml.g6.4xlarge` for Mini, `ml.g6.12xlarge` for Small

### NOT Available On:
- HuggingFace Inference Providers
- Together AI, Fireworks AI, Groq, Replicate
- Azure AI, GCP Vertex AI

---

## Provider Summary Table

| Provider | Model | Streaming | Pricing | Notes |
|---|---|---|---|---|
| **Mistral API (Realtime)** | Realtime 4B | ✅ WebSocket | $0.006/min | Best cloud streaming |
| **Mistral API (Batch)** | Transcribe V2 | ❌ | $0.003/min | Best batch quality |
| **Self-host vLLM** | Realtime 4B | ✅ WebSocket | GPU cost | Best local streaming |
| **voxtral.c** | Realtime 4B | ✅ C API | Hardware only | Lean, zero-dep |
| **llama.cpp GGUF** | Mini 3B | ⚠️ Chunked | Hardware only | Lowest HW requirements |
| **DeepInfra** | Mini 3B | ❌ | $0.001/min | Cheapest cloud |
| **AWS Bedrock** | Mini/Small | ❌ (chat) | Pay-per-use | Not dedicated STT |

---

## Linux Desktop Dictation: The Full Picture

Your goal: **speak → text appears wherever the cursor is**, system-wide.

Architecture:
```
[Hotkey] → [Audio Capture] → [STT Engine] → [Text Injection] → [Any App]
                pw-record       voxtral.c       ydotool type
                sounddevice     vLLM server     wtype
                parec           llama.cpp       clipboard+paste
```

### Existing Linux Dictation Tools

| Tool | Stars | STT Backend | Wayland | Pluggable Backend? |
|---|---|---|---|---|
| **[nerd-dictation](https://github.com/ideasman42/nerd-dictation)** | 1,744 | VOSK | ❌ (X11 via xdotool) | ⚠️ Hackable (single Python file) |
| **[hyprwhspr](https://github.com/goodroot/hyprwhspr)** | 791 | Parakeet, whisper.cpp, **REST API**, **WebSocket** | ✅ | ✅ **Excellent** — REST backend accepts any URL |
| **[Speech Note](https://github.com/mkiol/dsnote)** | 1,345 | Whisper, VOSK, DeepSpeech, Faster-Whisper | ✅ | ✅ Multiple built-in |
| **[voxtype](https://github.com/peteonrails/voxtype)** | 308 | whisper.cpp + remote server | ✅ Wayland-first | ✅ + LLM post-processing pipe |
| **[OpenWhispr](https://github.com/OpenWhispr/openwhispr)** | 1,043 | Whisper + Parakeet via sherpa-onnx | ✅ | ✅ |
| **[keyless](https://github.com/hate/keyless)** | 18 | Whisper (Candle, GGUF) | ✅ | ⚠️ Limited |

### Text Injection Methods

| Tool | X11 | Wayland (wlroots) | Wayland (GNOME) | Notes |
|---|---|---|---|---|
| **xdotool** | ✅ | ❌ | ❌ | Classic, X11 only |
| **ydotool** | ✅ | ✅ | ✅ | Universal (uinput). Needs `input` group + `ydotoold` |
| **dotool** | ✅ | ✅ | ✅ | Similar to ydotool, simpler |
| **wtype** | ❌ | ✅ | ❌ | Best for wlroots (Sway, Hyprland) |
| **clipboard+paste** | ✅ | ✅ | ✅ | `wl-copy`/`xclip` + Ctrl+V. Most reliable universal fallback |

voxtype's recommended fallback chain: `wtype → dotool → ydotool → clipboard paste`

### No One Has Built Voxtral + Linux Dictation Yet

**This would be novel.** The closest pieces are:
- **voxtral.c** provides streaming C inference
- **hyprwhspr** provides pluggable REST API backend + text injection
- The gap: connecting them

---

## Recommended Paths for Your Setup

| Your GPU VRAM | Recommended Stack | Expected Latency |
|---|---|---|
| **≥16 GB** (RTX 3090/4080/4090) | Voxtral Realtime 4B + vLLM → hyprwhspr (REST/WebSocket backend) | **<500ms** (true streaming) |
| **10-16 GB** (RTX 3060 12GB, 4060 Ti) | Voxtral Mini 3B + vLLM/llama.cpp + VAD chunking → ydotool | **1-3 seconds** |
| **<10 GB or CPU-only** | Voxtral Mini 3B Q4 GGUF + llama.cpp + VAD → ydotool | **2-5 seconds** |
| **Any (with internet)** | Mistral Realtime API ($0.006/min) → hyprwhspr | **<200ms** |
| **Apple Silicon** | voxtral.c (Metal) or MLX via mlx-audio | **TBD** |

### Fastest Path to Working Dictation

**Option A — Cloud, zero hardware (minutes to set up):**
1. Install `hyprwhspr`
2. Configure REST API backend pointing at Mistral's realtime WebSocket
3. Done — sub-200ms dictation everywhere

**Option B — Fully local with ≥16GB GPU (the dream):**
1. Start vLLM serving Voxtral Realtime 4B
2. Write a ~50-line Python script: mic → WebSocket → ydotool/wtype
3. Bind to hotkey
4. Completely private, no internet needed

**Option C — Fully local, minimal hardware:**
1. Run llama.cpp server with Voxtral Mini 3B Q4_K_M (2.5 GB)
2. Use Silero VAD to detect speech boundaries
3. Send completed utterances to llama.cpp via chat completions with audio
4. Inject via ydotool
5. ~2-5 second delay per utterance, but works on almost any machine

---

*Sources: mistral.ai/news/voxtral-transcribe-2, HuggingFace model cards, llama.cpp GitHub, antirez/voxtral.c, hyprwhspr, nerd-dictation, Mistral API docs*
