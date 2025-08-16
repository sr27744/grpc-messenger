<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { chatClient } from '$lib/chatClient';
  import type { StreamEnvelope } from '$lib/gen/chat/v1/chat_pb';

  let roomId = 'general';
  let text = '';
  let messages: { who: string; body: string }[] = [];
  let stopStream: null | (() => void) = null;

  async function loadHistory() {
    const resp = await chatClient.history({ roomId, limit: 50 });
    (resp.messages ?? []).forEach(m => {
      messages = [...messages, { who: m.sender?.displayName ?? 'user', body: m.text }];
    });
  }

  function handleEvent(ev: StreamEnvelope) {
    if (ev.payload?.case === 'message') {
      const m = ev.payload.value;
      messages = [...messages, { who: m.sender?.displayName ?? 'user', body: m.text }];
    }
  }

  async function startStream() {
    const s = chatClient.chatStream({ roomId, limit: 50 });
    (async () => {
      try {
        for await (const ev of s) handleEvent(ev);
      } catch (err) {
        console.error('stream error', err);
      }
    })();
    stopStream = () => (s as any).close?.();
  }

  async function send() {
    const t = text.trim();
    if (!t) return;
    await chatClient.send({ roomId, text: t });
    text = '';
  }

  onMount(async () => {
    await loadHistory();
    await startStream();
  });

  onDestroy(() => stopStream?.());
</script>

<svelte:head>
  <title>gRPC Chat (SvelteKit)</title>
</svelte:head>

<main style="max-width:640px;margin:2rem auto;font-family:system-ui,sans-serif;">
  <h2>gRPC Chat — room: {roomId}</h2>

  <div style="border:1px solid #ccc;height:320px;overflow:auto;padding:.5rem;margin:.75rem 0;">
    {#each messages as m}
      <div><strong>{m.who}:</strong> {m.body}</div>
    {/each}
  </div>

  <input
    placeholder="Type a message…"
    bind:value={text}
    on:keydown={(e) => e.key === 'Enter' && send()}
    style="width:100%;padding:.6rem;margin-bottom:.5rem;"
  />
  <button on:click={send}>Send</button>
</main>

