import { errorMessage } from './pb';

// Letters and digits that can't be mistaken for each other (no i, l, o, 0 or 1),
// in lower case so the password is easy to type on a phone.
export const UNAMBIGUOUS_CHARS = 'abcdefghjkmnpqrstuvwxyz23456789';

/**
 * Makes a password for someone the owner adds: 4 groups of 4, like
 * "k7mq-x3vb-9hfe-2ntw" (about 79 bits). Draws again on the few bytes that
 * would make some letters more likely than others.
 */
export function generatePassword(
  randomBytes: (n: number) => Uint8Array = (n) => crypto.getRandomValues(new Uint8Array(n)),
): string {
  const n = UNAMBIGUOUS_CHARS.length;
  const limit = 256 - (256 % n);
  let chars = '';
  while (chars.length < 16) {
    for (const byte of randomBytes(32)) {
      if (byte < limit && chars.length < 16) chars += UNAMBIGUOUS_CHARS[byte % n];
    }
  }
  return chars.match(/.{4}/g)!.join('-');
}

export interface Invite {
  appName: string;
  address: string;
  email: string;
  password: string;
  name?: string;
}

/** The message the owner sends to someone they added, or whose password they reset. */
export function inviteText({ appName, address, email, password, name }: Invite): string {
  return [
    `${name ? `Hi ${name}, here` : 'Here'} is your account for ${appName}.`,
    '',
    `Open: ${address}`,
    `Email: ${email}`,
    `Password: ${password}`,
    '',
    'Change it after you sign in.',
  ].join('\n');
}

/** Turns an error from saving a person into a sentence to show the owner. */
export function personError(err: unknown): string {
  const data = (err as { response?: { data?: Record<string, { code?: string }> } })?.response?.data;
  if (data?.email?.code === 'validation_not_unique') return 'Someone already uses that email.';
  if (data?.email) return 'Check the email address.';
  return errorMessage(err);
}
