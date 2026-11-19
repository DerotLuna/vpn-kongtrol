import { describe, expect, it, vi } from 'vitest'
import { announceReady, type TrayMessage } from './bridge'

// A document whose visibility the test controls.
function fakeDoc(state: DocumentVisibilityState) {
  const target = new EventTarget()
  const doc = Object.assign(target, { visibilityState: state }) as unknown as Document & {
    visibilityState: DocumentVisibilityState
  }
  const set = (s: DocumentVisibilityState) => {
    ;(doc as { visibilityState: DocumentVisibilityState }).visibilityState = s
    doc.dispatchEvent(new Event('visibilitychange'))
  }
  return { doc, set }
}

const now = (cb: () => void) => cb()

describe('announceReady', () => {
  it('posts ready at once when the page is visible', () => {
    const post = vi.fn<(m: TrayMessage) => void>()
    const { doc } = fakeDoc('visible')
    announceReady({ post }, doc, now)
    expect(post.mock.calls).toEqual([['ready']])
  })

  it('waits while the page is hidden and posts once when it becomes visible', () => {
    const post = vi.fn<(m: TrayMessage) => void>()
    const { doc, set } = fakeDoc('hidden')
    announceReady({ post }, doc, now)
    expect(post).not.toHaveBeenCalled()
    set('visible')
    set('hidden')
    set('visible')
    expect(post.mock.calls).toEqual([['ready']])
  })

  it('posts after a painted frame, not before', () => {
    const post = vi.fn<(m: TrayMessage) => void>()
    const frames: (() => void)[] = []
    const { doc } = fakeDoc('visible')
    announceReady({ post }, doc, (cb) => frames.push(cb))
    expect(post).not.toHaveBeenCalled()
    frames.forEach((f) => f())
    expect(post.mock.calls).toEqual([['ready']])
  })

  it('cleanup cancels a pending announcement', () => {
    const post = vi.fn<(m: TrayMessage) => void>()
    const { doc, set } = fakeDoc('hidden')
    const cancel = announceReady({ post }, doc, now)
    cancel()
    set('visible')
    expect(post).not.toHaveBeenCalled()
  })
})
