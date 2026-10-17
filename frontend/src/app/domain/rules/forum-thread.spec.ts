import { ForumPost } from '../models';
import { isSolved, threadReplies } from './forum-thread';

const post = (over: Partial<ForumPost> = {}): ForumPost => ({
  id: 'p1',
  authorId: 'asker',
  category: 'health',
  title: 'Vaccin pour poussins ?',
  body: 'Quel calendrier ?',
  createdAt: '2026-10-17T10:00:00Z',
  replies: [
    { authorId: 'a', body: 'first', createdAt: '2026-10-17T10:05:00Z' },
    { authorId: 'asker', body: 'thanks', createdAt: '2026-10-17T10:06:00Z' },
    { authorId: 'b', body: 'the answer', createdAt: '2026-10-17T10:07:00Z' }
  ],
  ...over
});

describe('threadReplies', () => {
  it('keeps the replies in order when none is the best', () => {
    const replies = threadReplies(post());
    expect(replies.map((r) => r.reply.body)).toEqual(['first', 'thanks', 'the answer']);
    expect(replies.some((r) => r.best)).toBeFalse();
  });

  it('pins the best answer first, keeping its index for the API', () => {
    const replies = threadReplies(post({ bestReplyIndex: 2 }));
    expect(replies.map((r) => r.reply.body)).toEqual(['the answer', 'first', 'thanks']);
    expect(replies[0]).toEqual(jasmine.objectContaining({ index: 2, best: true }));
  });

  it("tags the asker's own replies", () => {
    expect(threadReplies(post()).map((r) => r.byAuthor)).toEqual([false, true, false]);
  });

  it('has nothing to show for a thread without replies', () => {
    expect(threadReplies(post({ replies: undefined }))).toEqual([]);
  });
});

describe('isSolved', () => {
  it('is solved once a best answer is marked, the first one included', () => {
    expect(isSolved(post())).toBeFalse();
    expect(isSolved(post({ bestReplyIndex: 0 }))).toBeTrue();
  });
});
