// Function to derive a key from the shared password
async function deriveKey(password) {
  const encoder = new TextEncoder();
  const salt = crypto.getRandomValues(new Uint8Array(16)); // This salt should be stored or sent along with the encrypted data
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    encoder.encode(password),
    { name: "PBKDF2" },
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

// Function to encrypt data
async function encryptData(plainText, key) {
  const iv = crypto.getRandomValues(new Uint8Array(12)); // Generate a new IV for each encryption
  const encrypted = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv },
    key,
    new TextEncoder().encode(plainText)
  );
  return { encrypted, iv };
}

// Function to decrypt data
async function decryptData(encryptedData, key, iv) {
  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: iv },
    key,
    encryptedData
  );
  return new TextDecoder().decode(decrypted);
}

async function encryptData(plainText, password) {
  const encoder = new TextEncoder();
  // Generate a new salt for each encryption session
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    encoder.encode(password),
    { name: "PBKDF2" },
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt"]
  );
  const iv = crypto.getRandomValues(new Uint8Array(12)); // Initialization Vector
  const encrypted = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    encoder.encode(plainText)
  );
  return {
    encrypted: new Uint8Array(encrypted),
    iv,
    salt, // Include the salt in the output
  };
}

async function decryptData(encryptedData, password, salt, iv) {
  const encoder = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    encoder.encode(password),
    { name: "PBKDF2" },
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["decrypt"]
  );
  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    key,
    encryptedData
  );
  return new TextDecoder().decode(decrypted);
}

async function encryptPrivateKey(privateKey, password) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt"]
  );
  const exportedPrivateKey = await crypto.subtle.exportKey("pkcs8", privateKey);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encrypted = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    exportedPrivateKey
  );
  return { encrypted, iv, salt };
}

async function decryptPrivateKey(encryptedData, password, iv, salt) {
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["decrypt"]
  );
  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    key,
    encryptedData
  );
  const privateKey = await crypto.subtle.importKey(
    "pkcs8",
    decrypted,
    { name: "RSA-OAEP", hash: "SHA-256" },
    true,
    ["decrypt"]
  );
  return privateKey;
}

/*
Approach to ok-ish e2ee

1. Generate key pairs in FE
2. Save private key securely in FE local storage
3. Encrypt private key and save to BE (todo for later, encrypt that field at the DB-level)
4. Committed pairs need to exchange their public keys over the BE (FE->BE->FE)
5. Committed pairs can encrypt and decrypt messages from the pair chat.


Update password

1. Retrieve encrypted private key and decrypt with password

Password lost

1. Validate and trust public keys received from the server


Simplified Key Recovery Process
1. Key Pair Generation on Account Setup
Each user creates their own key pair when they set up their account. The public key is shared and stored on the server, accessible to their partner, while the private key is stored securely on the user's device.

2. Encryption of Messages
Sending Messages: When a user sends a message, they encrypt it with the recipient's public key. This ensures that only the intended recipient can decrypt the message with their private key.
Optional: For backup or synchronization purposes across multiple devices, the user can also encrypt the message with their own public key.
3. Password Reset and Key Regeneration
Reset Process: If a user forgets their password and needs to reset it, part of the reset process includes generating a new key pair.
Notification: Upon generating a new key pair, the system automatically notifies the partner that the user has reset their key pair and needs their messages re-encrypted with the new public key.
4. Granting Access for Recovery
Partner's Role: Upon receiving the notification, the partner can choose to grant or refuse the request to decrypt and re-encrypt the messages.
Re-encryption Process: If granted, the partner decrypts any messages they previously encrypted for the user with the old public key and re-encrypts them using the new public key. This data is then securely transmitted back to the user.
5. Recovering the Messages
The user who reset their password can now decrypt the re-encrypted messages using their new private key.

TODO for later:
Backup Options: Consider providing users with the option to back up their encrypted data locally. This backup can be secured with a passphrase or recovery mechanism that the user controls, independent of the system’s primary encryption keys.
*/
async function encryptPrivateKey(privateKey, password) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt"]
  );
  const exportedPrivateKey = await crypto.subtle.exportKey("pkcs8", privateKey);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encrypted = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    exportedPrivateKey
  );
  return { encrypted, iv, salt };
}

async function decryptPrivateKey(encryptedData, password, iv, salt) {
  const keyMaterial = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(password),
    "PBKDF2",
    false,
    ["deriveKey"]
  );
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt, iterations: 100000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["decrypt"]
  );
  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    key,
    encryptedData
  );
  const privateKey = await crypto.subtle.importKey(
    "pkcs8",
    decrypted,
    { name: "RSA-OAEP", hash: "SHA-256" },
    true,
    ["decrypt"]
  );
  return privateKey;
}
