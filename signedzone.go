package goresolver

import (
	"github.com/miekg/dns"
	"log"
	"strings"
	"time"
)

type SignedZone struct {
	zone        string
	dnskey      SignedRRSet
	ds          SignedRRSet
	signingKeys map[uint16]*dns.DNSKEY
	parentZone  *SignedZone
}

func (z SignedZone) getKeyByTag(keyTag uint16) *dns.DNSKEY {
	return z.signingKeys[keyTag]
}

func (z SignedZone) addSigningKey(k *dns.DNSKEY) {
	z.signingKeys[k.KeyTag()] = k
}

func (z SignedZone) validateRRSIG(sig *dns.RRSIG, rrSet []dns.RR) (err error) {
	// Verify the RRSIG of the DNSKEY RRset
	key := z.getKeyByTag(sig.KeyTag)
	if key == nil {
		log.Printf("DNSKEY keytag %d not found", sig.KeyTag)
		return ErrDnskeyNotAvailable
	}
	err = sig.Verify(key, rrSet)

	if err != nil {
		log.Printf("validation DNSKEY: %s\n", err)
		return err
	}

	if sig.ValidityPeriod(time.Now()) == false {
		log.Printf("invalid validity period on signature: %s\n", err)
		return ErrRrsigValidityPeriod
	}
	return nil
}

func (z SignedZone) validateDS(dsRrset []dns.RR) (err error) {

	for _, rr := range dsRrset {

		ds := rr.(*dns.DS)

		if ds.DigestType != dns.SHA256 {
			log.Printf("Unknown digest type (%d) on DS RR", ds.DigestType)
			continue
		}

		parentDsDigest := strings.ToUpper(ds.Digest)
		key := z.getKeyByTag(ds.KeyTag)
		if key == nil {
			log.Printf("DNSKEY keytag %d not found", ds.KeyTag)
			return ErrDnskeyNotAvailable
		}
		dsDigest := strings.ToUpper(key.ToDS(ds.DigestType).Digest)
		if parentDsDigest == dsDigest {
			return nil
		}

		log.Printf("DS does not match DNSKEY\n")
		return ErrDsInvalid
	}
	return ErrUnknownDsDigestType
}
