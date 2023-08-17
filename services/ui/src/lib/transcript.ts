export interface TranscriptWord {
  start: number
  end: number
  word: string
  probability: number
}

export interface TranscriptSegment {
  id: string
  seek: number // 10ms frame
  start: number
  end: number
  temperature: number
  avg_logprob: number
  compression_ratio: number
  no_speech_prob: number

  text?: string
  words: TranscriptWord[]
}

export interface Transcript {
  language: string
  language_probability: number
  endTimestamp: number
  duration: number
  segments: TranscriptSegment[]
}

export interface TranscriptDocument {
  transcriptions: Transcript[]
	transcribedText: string
	currentTranscription: string
	newText: string
  startedAt: number
}

export interface RenderedTranscriptSession {
  participants: string[]
  related: string
  statistics: string
  summary: string
  headline: string
  date: Date
  entries: RenderedTranscriptEntry[]
}

export interface RenderedTranscriptEntry {
  time: Date
  isAssistant: boolean
  precedingSilence: number
  sessionTime: number
  speakerLabel: string
  text: string
  debug: {
    precedingSilence: number
    sessionTimeMs: number
    sessionTime: number
    transcriptEndTimestamp: number
    transcriptStartTimestamp: number
    transcript: Transcript
    segment: TranscriptSegment
  }[]
}

export function renderableTranscriptSession(doc: TranscriptDocument): RenderedTranscriptSession {
  const speakerLabel = 'Unknown'
  const isAssistant = false
  const session: RenderedTranscriptSession = {
    participants: [speakerLabel],
    related: '',
    statistics: '',
    summary: '',
    headline: '',
    date: new Date(),
    entries: [],
  }

  if (!doc) {
    return session
  }

  const startedAtMs = doc.startedAt * 1000
  session.date = new Date(startedAtMs)

  let lastSegmentSessionEndTime
  let lastEntry: RenderedTranscriptEntry | undefined
  for (const transcript of doc.transcriptions) {
    const transcriptEndTimestamp = transcript.endTimestamp / 1000
    const transcriptStartTimestamp = transcript.endTimestamp - transcript.duration
    for (const segment of transcript.segments) {
      const sessionTimeMs = segment.start + transcriptStartTimestamp
      const sessionTime = Math.floor(sessionTimeMs / 1000)
      const precedingSilence = lastSegmentSessionEndTime == null ? sessionTime : sessionTime - lastSegmentSessionEndTime

      if (lastEntry &&
          lastEntry.speakerLabel === speakerLabel &&
          lastEntry.isAssistant === isAssistant &&
          precedingSilence < 2) {
        lastEntry.text += segment.text || ''
        lastEntry.debug.push({ precedingSilence, transcript, sessionTimeMs, sessionTime, transcriptEndTimestamp, transcriptStartTimestamp, segment })
      } else {
        lastEntry = {
          speakerLabel,
          isAssistant,
          precedingSilence,
          sessionTime,
          time: new Date(startedAtMs + sessionTimeMs),
          text: segment.text || '',
          debug: [{ precedingSilence, transcript, sessionTimeMs, sessionTime, transcriptEndTimestamp, transcriptStartTimestamp, segment}]
        }
        session.entries.push(lastEntry)
      }
      lastSegmentSessionEndTime = transcriptStartTimestamp + segment.end 
    }
    lastSegmentSessionEndTime = transcriptEndTimestamp
  }

  return session
}