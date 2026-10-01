import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getLocations, getFiles, getFileUrl, isBlueFile } from '../api/sds';

describe('SDS API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  describe('getLocations', () => {
    it('returns location names from API', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { location_name: 'TestDir' },
              { location_name: 'minio' },
            ]),
        })
      );

      const result = await getLocations();
      expect(result).toEqual(['TestDir', 'minio']);
      expect(fetch).toHaveBeenCalledWith('/sds/fs');
    });

    it('throws on request failure', async () => {
      global.fetch = vi.fn(() => Promise.reject(new Error('network')));
      await expect(getLocations()).rejects.toThrow('network');
    });
  });

  describe('getFiles', () => {
    it('fetches files for a location', async () => {
      const files = [
        { filename: 'test.tmp', type: 'file' },
        { filename: 'subdir', type: 'directory' },
      ];
      global.fetch = vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(files) })
      );

      const result = await getFiles('TestDir');
      expect(result).toEqual(files);
      expect(fetch).toHaveBeenCalledWith('/sds/fs/TestDir/');
    });

    it('includes encoded path segments when provided', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      );

      await getFiles('MinIO #1', 'sub dir/a#b?c%25|d');
      expect(fetch).toHaveBeenCalledWith(
        '/sds/fs/MinIO%20%231/sub%20dir/a%23b%3Fc%2525%7Cd'
      );
    });

    it('turns null listings into an empty array', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(null) })
      );

      await expect(getFiles('TestDir')).resolves.toEqual([]);
    });

    it('throws on http error', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({ ok: false, status: 500 })
      );
      await expect(getFiles('TestDir')).rejects.toThrow('500');
    });
  });

  describe('getFileUrl', () => {
    it('builds encoded URL for fs mode', () => {
      expect(getFileUrl('data #1.tmp', 'fs', 'Test Dir')).toBe(
        '/sds/fs/Test%20Dir/data%20%231.tmp'
      );
    });

    it('encodes reserved path characters for sigplot hrefs', () => {
      expect(getFileUrl('a/b?c%25|d', 'hdr', 'MinIO #1')).toBe(
        '/sds/hdr/MinIO%20%231/a/b%3Fc%2525%7Cd'
      );
    });
  });

  describe('isBlueFile', () => {
    it('accepts the extensions SDS can serve', () => {
      expect(isBlueFile('data.tmp')).toBe(true);
      expect(isBlueFile('penny.prm')).toBe(true);
    });

    it('rejects everything else', () => {
      expect(isBlueFile('main.go')).toBe(false);
      expect(isBlueFile('config.json')).toBe(false);
      expect(isBlueFile('data.tmp.bak')).toBe(false);
      expect(isBlueFile('tmp')).toBe(false);
    });
  });
});
