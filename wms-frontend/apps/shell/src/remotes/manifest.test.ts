import { describe, expect, it } from 'vitest';
import { allRemotes, getEnabledRemoteNames, getEnabledRemotes } from './manifest';

describe('remote manifest helpers', () => {
  it('returns all remotes when env value is empty', () => {
    expect(getEnabledRemoteNames(undefined)).toHaveLength(allRemotes.length);
    expect(getEnabledRemotes(undefined)).toHaveLength(allRemotes.length);
  });

  it('filters out unknown remote names gracefully', () => {
    const enabledNames = getEnabledRemoteNames('orders,unknown,waves');
    expect(enabledNames).toEqual(['orders', 'waves']);
  });

  it('returns remote definitions that respect the enabled order', () => {
    const remotes = getEnabledRemotes('waves,orders');
    expect(remotes.map((remote) => remote.name)).toEqual(['waves', 'orders']);
  });
});
