import { cycleStatusEntry, escrowStatusEntry, priorityStatusEntry, trustScoreTone } from './status-registry';

describe('status-registry', () => {
  it('gives every CycleStatus its fixed label key and tone', () => {
    expect(cycleStatusEntry('planned')).toEqual({ labelKey: 'status.cyclePlanned', tone: 'sky' });
    expect(cycleStatusEntry('in_progress')).toEqual({ labelKey: 'status.cycleInProgress', tone: 'forest' });
    expect(cycleStatusEntry('done')).toEqual({ labelKey: 'status.cycleDone', tone: 'stone' });
    expect(cycleStatusEntry('cancelled')).toEqual({ labelKey: 'status.cycleCancelled', tone: 'alert' });
  });

  it('gives every escrow PaymentStatus its fixed label key and tone', () => {
    expect(escrowStatusEntry('held')).toEqual({ labelKey: 'status.escrowHeld', tone: 'sky' });
    expect(escrowStatusEntry('refunded')).toEqual({ labelKey: 'status.escrowRefunded', tone: 'plum' });
  });

  it('reads a paid reservation as prioritaire, forest', () => {
    expect(priorityStatusEntry(true).tone).toBe('forest');
    expect(priorityStatusEntry(false).tone).toBe('amber');
  });

  describe('trustScoreTone', () => {
    it('is forest at 80 and above', () => {
      expect(trustScoreTone(80)).toBe('forest');
      expect(trustScoreTone(100)).toBe('forest');
    });

    it('is amber between 60 and 79', () => {
      expect(trustScoreTone(60)).toBe('amber');
      expect(trustScoreTone(79)).toBe('amber');
    });

    it('is alert below 60', () => {
      expect(trustScoreTone(0)).toBe('alert');
      expect(trustScoreTone(59)).toBe('alert');
    });
  });
});
