import { ForumPost, ForumReply } from '../models';

/** One reply with its place in the thread: the API marks the best
 * answer by its index among the replies. */
export interface ThreadReply {
  readonly reply: ForumReply;
  readonly index: number;
  readonly best: boolean;
  /** Written by whoever asked the question. */
  readonly byAuthor: boolean;
}

/** A thread's replies as the page shows them: the best answer first,
 * then the others in the order they came. */
export function threadReplies(post: ForumPost): ThreadReply[] {
  const all = (post.replies ?? []).map((reply, index) => ({
    reply,
    index,
    best: post.bestReplyIndex === index,
    byAuthor: reply.authorId === post.authorId
  }));
  return [...all.filter((r) => r.best), ...all.filter((r) => !r.best)];
}

/** Whether the asker marked an answer as the best. */
export function isSolved(post: ForumPost): boolean {
  return post.bestReplyIndex !== undefined && post.bestReplyIndex !== null;
}
