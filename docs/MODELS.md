# Voxtral Model Reference

*Last updated: February 2026*

Mistral's speech-to-text models are called **Voxtral**. All open models are
Apache 2.0 licensed.

## Model Matrix

| Model | Params | VRAM (bf16) | VRAM (Q4) | Use Case | Streaming |
|---|---|---|---|---|---|
| **Voxtral Mini 3B** | ~5B | 9.5 GB | 2.5 GB | Edge/desktop, general transcription | No (batch) |
| **Voxtral Small 24B** | 24B | 55 GB | 14 GB | Premium quality transcription | No (batch) |
| **Voxtral Realtime 4B** | ~4B | 16 GB | N/A* | Live streaming transcription | **Yes** |
| Voxtral Mini Transcribe V2 | ~4B | API only | API only | Best batch transcription | No |

\* Realtime 4B has no quantized versions yet (novel architecture, only vLLM supported).

## Architecture

All Voxtral models are encoder-decoder:
- **Audio encoder**: Whisper-style, processes 16kHz audio into embeddings
- **Language model**: Mistral-family transformer, generates text from embeddings

The Realtime 4B uses a **causal** audio encoder (processes audio left-to-right
as it arrives), while the offline models use bidirectional attention.

## Quality Benchmarks (WER, lower = better)

| Model | English short | English long | Multilingual (FLEURS) |
|---|---|---|---|
| Voxtral Small 24B | ~2.5% | ~5% | ~4% |
| Voxtral Mini 3B | ~3.5% | ~7% | ~6% |
| Voxtral Realtime 4B @480ms | ~5% | ~10% | ~9% |
| Voxtral Realtime 4B @2.4s | ~4% | ~7% | ~7% |
| Whisper large-v3 | ~4% | ~8% | ~8% |
| GPT-4o mini Transcribe | ~3% | ~6% | ~5% |

Voxtral outperforms Whisper across all tasks. Small 24B matches or beats GPT-4o.

## Serving Frameworks

| Framework | Mini 3B | Small 24B | Realtime 4B |
|---|---|---|---|
| **vLLM** (≥0.10.0) | ✅ | ✅ | ✅ (only option) |
| **llama.cpp** (≥b6014) | ✅ (GGUF) | ✅ (GGUF) | ❌ Not yet |
| **Transformers** (≥4.54.0) | ✅ | ✅ | ❌ Not yet |
| **MLX** (mlx-audio) | ✅ | ✅ (8-bit) | ❌ |

## GGUF Quant Sources (for llama.cpp)

| Source | Mini 3B | Small 24B | Notes |
|---|---|---|---|
| bartowski (HuggingFace) | Full range BF16→IQ2_M | Full range | Most popular, imatrix |
| ggml-org (HuggingFace) | Q4_K_M + mmproj | — | Official |
| mradermacher (HuggingFace) | Standard + imatrix | Standard + imatrix | Alternative |

**Important:** llama.cpp needs TWO files: the main model GGUF + the mmproj
(audio encoder) GGUF. The mmproj is always ~700MB at Q8.

## Cloud API Providers

| Provider | Model | Streaming | Price |
|---|---|---|---|
| **Mistral API** | Realtime | ✅ WebSocket | $0.006/min |
| **Mistral API** | Transcribe V2 | ❌ Batch | $0.003/min |
| **DeepInfra** | Mini 3B | ❌ Batch | $0.001/min |
| Others (Groq, Together, etc.) | ❌ | ❌ | Not available yet |

## Other Local Inference Options

- **voxtral.c** by antirez: Pure C inference for Realtime 4B, zero dependencies,
  streaming C API. Metal (Apple) + OpenBLAS backends. github.com/antirez/voxtral.c
- **RedHatAI FP8**: 50% VRAM reduction for vLLM. On HuggingFace.
- **ONNX**: Browser/WebGPU inference. `onnx-community/Voxtral-Mini-3B-2507-ONNX`

## Key Links

- Blog: https://mistral.ai/news/voxtral-transcribe-2
- Paper: https://arxiv.org/abs/2507.13264
- API docs: https://docs.mistral.ai/capabilities/audio/
- Mini 3B: https://huggingface.co/mistralai/Voxtral-Mini-3B-2507
- Small 24B: https://huggingface.co/mistralai/Voxtral-Small-24B-2507
- Realtime 4B: https://huggingface.co/mistralai/Voxtral-Mini-4B-Realtime-2602
