import { describe, expect, it } from 'vitest';
import { UNAMBIGUOUS_CHARS, generatePassword, inviteText, personError } from './people';

describe('generatePassword', () => {
  it('makes 4 groups of 4 from the alphabet', () => {
    for (let i = 0; i < 200; i++) {
      expect(generatePassword()).toMatch(/^[a-z2-9]{4}-[a-z2-9]{4}-[a-z2-9]{4}-[a-z2-9]{4}$/);
    }
  });

  it('never uses look-alike characters', () => {
    const all = Array.from({ length: 200 }, () => generatePassword()).join('');
    for (const c of 'ilo01IO') expect(all).not.toContain(c);
    for (const c of all.replaceAll('-', '')) expect(UNAMBIGUOUS_CHARS).toContain(c);
  });

  it('skips bytes that would favor some characters', () => {
    // 248 and above are skipped (256 % 31 = 8), so only the 3s and 0s count
    const bytes = (n: number) =>
      Uint8Array.from({ length: n }, (_, i) => (i % 2 ? 250 : i % 4 ? 3 : 0));
    expect(generatePassword(bytes)).toBe('adad-adad-adad-adad');
  });

  it('differs every time', () => {
    expect(new Set(Array.from({ length: 50 }, () => generatePassword())).size).toBe(50);
  });
});

describe('inviteText', () => {
  const invite = {
    appName: 'plasto',
    address: 'https://home.example.com',
    email: 'sam@example.com',
    password: 'abcd-efgh-jkmn-pqrs',
  };

  it('has everything they need to sign in', () => {
    expect(inviteText({ ...invite, name: 'Sam' })).toBe(
      [
        'Hi Sam, here is your account for plasto.',
        '',
        'Open: https://home.example.com',
        'Email: sam@example.com',
        'Password: abcd-efgh-jkmn-pqrs',
        '',
        'Change it after you sign in.',
      ].join('\n'),
    );
  });

  it('works without a name', () => {
    expect(inviteText(invite)).toMatch(/^Here is your account for plasto\./);
  });
});

describe('personError', () => {
  it('explains an email that is already used', () => {
    const err = { status: 400, response: { data: { email: { code: 'validation_not_unique' } } } };
    expect(personError(err)).toBe('Someone already uses that email.');
  });

  it("falls back to the server's message", () => {
    expect(personError({ status: 403, response: { message: 'Nope.' } })).toBe('Nope.');
  });
});
