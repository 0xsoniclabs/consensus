// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

// BaseDescriptor identifies a base event by its creator and hash.
type BaseDescriptor struct {
	ValidatorID ValidatorID
	BaseHash    EventHash
}
