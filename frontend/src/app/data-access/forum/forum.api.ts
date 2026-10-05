import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ForumPost } from '../../domain/models';
import { ApiClient, QueryParams } from '../../core/http/api-client';

export interface ForumSearchParams {
  readonly category?: string;
  readonly q?: string;
  readonly sort?: string;
}

export interface CreateForumPostInput {
  readonly category: string;
  readonly title: string;
  readonly body: string;
  readonly listingId?: string;
}

@Injectable({ providedIn: 'root' })
export class ForumApi {
  private readonly api = inject(ApiClient);

  postsUrl(params: ForumSearchParams = {}): string {
    return this.api.url('/forum/posts', params as QueryParams);
  }

  postUrl(id: string): string {
    return this.api.url(`/forum/posts/${id}`);
  }

  create(input: CreateForumPostInput): Observable<ForumPost> {
    return this.api.post<ForumPost>('/forum/posts', input);
  }

  reply(postId: string, body: string): Observable<ForumPost> {
    return this.api.post<ForumPost>(`/forum/posts/${postId}/replies`, { body });
  }

  markBestAnswer(postId: string, index: number): Observable<ForumPost> {
    return this.api.post<ForumPost>(`/forum/posts/${postId}/replies/${index}/mark-best`);
  }
}
