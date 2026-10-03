import { OpenFlareBaseService } from './base.service';
import type {
  OriginDetail,
  OriginHealth,
  OriginItem,
  OriginMutationPayload,
} from './types';

export class OriginService extends OpenFlareBaseService {
  protected static override readonly basePath: string = '/api/v1/d/origins';

  static list(): Promise<OriginItem[]> {
    return this.get<OriginItem[]>('/');
  }

  static getById(id: number): Promise<OriginDetail> {
    return this.get<OriginDetail>(`/${id}`);
  }

  static health(id: number): Promise<OriginHealth> {
    return this.get<OriginHealth>(`/${id}/health`);
  }

  static create(payload: OriginMutationPayload): Promise<OriginItem> {
    return this.post<OriginItem>('/', payload);
  }

  static update(
    id: number,
    payload: OriginMutationPayload,
  ): Promise<OriginItem> {
    return this.post<OriginItem>(`/${id}/update`, payload);
  }

  static deleteById(id: number): Promise<void> {
    return this.post<void>(`/${id}/delete`);
  }
}
