import { useState, useRef } from 'react';
import { fetchApi } from '../../api';
import { Send, Mic, Paperclip, Square } from 'lucide-react';
import { toast } from 'react-hot-toast';

export default function MessageInput({ sessionId, jid }) {
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [recording, setRecording] = useState(false);
  const mediaRecorderRef = useRef(null);
  const chunksRef = useRef([]);
  const fileInputRef = useRef(null);

  const sendText = async () => {
    if (!text.trim() || sending) return;
    setSending(true);
    try {
      await fetchApi(`/agent/chats/${encodeURIComponent(jid)}/send/text`, {
        method: 'POST',
        body: JSON.stringify({ session_id: sessionId, text: text.trim() }),
      });
      setText('');
    } catch (e) {
      toast.error('Failed to send: ' + e.message);
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendText();
    }
  };

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mr = new MediaRecorder(stream, { mimeType: 'audio/webm' });
      chunksRef.current = [];
      mr.ondataavailable = e => chunksRef.current.push(e.data);
      mr.onstop = async () => {
        const blob = new Blob(chunksRef.current, { type: 'audio/webm' });
        stream.getTracks().forEach(t => t.stop());
        await sendMedia(blob, 'audio', 'voice.webm');
      };
      mr.start();
      mediaRecorderRef.current = mr;
      setRecording(true);
    } catch (e) {
      toast.error('Microphone access denied');
    }
  };

  const stopRecording = () => {
    mediaRecorderRef.current?.stop();
    setRecording(false);
  };

  const sendMedia = async (blob, mediaType, filename) => {
    setSending(true);
    try {
      const fd = new FormData();
      fd.append('session_id', sessionId);
      fd.append('media_type', mediaType);
      fd.append('filename', filename);
      fd.append('file', blob, filename);
      await fetchApi(`/agent/chats/${encodeURIComponent(jid)}/send/media`, {
        method: 'POST',
        body: fd,
      });
      toast.success('Sent!');
    } catch (e) {
      toast.error('Failed to send: ' + e.message);
    } finally {
      setSending(false);
    }
  };

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // Detect type
    let mediaType = 'document';
    if (file.type.startsWith('image/')) mediaType = 'image';
    else if (file.type.startsWith('video/')) mediaType = 'video';
    else if (file.type.startsWith('audio/')) mediaType = 'audio';
    await sendMedia(file, mediaType, file.name);
    e.target.value = '';
  };

  return (
    <div className="message-input-bar">
      <input
        ref={fileInputRef}
        type="file"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      <button
        className="icon-btn"
        title="Attach file"
        onClick={() => fileInputRef.current?.click()}
        disabled={sending || recording}
      >
        <Paperclip size={20} />
      </button>

      <textarea
        className="message-input-text"
        placeholder="Type a message…"
        value={text}
        onChange={e => setText(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={1}
        disabled={sending || recording}
      />

      {recording ? (
        <button className="icon-btn icon-btn--recording" onClick={stopRecording} title="Stop recording">
          <Square size={20} />
        </button>
      ) : text.trim() ? (
        <button className="icon-btn icon-btn--send" onClick={sendText} disabled={sending} title="Send">
          <Send size={20} />
        </button>
      ) : (
        <button className="icon-btn" onClick={startRecording} disabled={sending} title="Record voice note">
          <Mic size={20} />
        </button>
      )}
    </div>
  );
}
