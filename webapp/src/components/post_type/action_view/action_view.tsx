import React from 'react';

import ActionButton from '@/components/post_type/action_view/action_button';
import type {Attachment, AttachmentAction, PollMetadata, PollMetadataMap, PollPost} from '@/types/poll';
import {ActionButtonType} from '@/utils/constants';

type Props = {
    post: PollPost;
    attachment: Attachment;
    pollMetadata?: PollMetadataMap;
    siteUrl: string;

    actions: {
        fetchPollMetadata: (siteUrl: string, pollId?: string) => void;
    };
};

export default class ActionView extends React.PureComponent<Props> {
    componentDidMount() {
        this.props.actions.fetchPollMetadata(this.props.siteUrl, this.props.post.props.poll_id);
    }

    /**
     * return true if the user has permission for adding option. if not, return false.
     * In details, return true in the following cases
     * - '--public-add-option' is set
     * or
     * - '--public-add-option' is NOT set AND can manage the poll
     * @param metadata metadata for poll
     * @return which or not the button for add option display
     */
    hasPermissionForAddOption(metadata: Partial<PollMetadata>): boolean {
        if (!metadata) {
            return false;
        }
        if (metadata.setting_public_add_option === true) {
            return true;
        }
        return Boolean(metadata.can_manage_poll);
    }

    /**
     * return true if the user has already voted the option named by `name`.
     * @param action the attachment action backing the button
     * @param metadata metadata for poll
     * @return voted or not
     */
    hasVoted(action: AttachmentAction, metadata: Partial<PollMetadata>): boolean {
        const votedAnswers = metadata.voted_answers;
        if (this.isAddOptionAction(action) || this.isPollManagementAction(action) || !votedAnswers) {
            return false;
        }
        const name = metadata.setting_progress ? action.name.replace(/ \([0-9]+\)$/, '') : action.name;
        return votedAnswers.indexOf(name) >= 0;
    }

    isAddOptionAction(action: AttachmentAction): boolean {
        return Boolean(action) && (action.id === 'addOption');
    }

    isPollManagementAction(action: AttachmentAction): boolean {
        return Boolean(action) && (action.id === 'endPoll' || action.id === 'deletePoll');
    }

    render() {
        const actions = this.props.attachment.actions;
        if (!actions || !actions.length) {
            return '';
        }

        const content: React.ReactNode[] = [];
        const metadataMap = this.props.pollMetadata || {};
        const pollId = this.props.post.props.poll_id;
        const metadata: Partial<PollMetadata> = (pollId ? metadataMap[pollId] : null) || {};

        actions.
            filter((action) => action.id && action.name).
            forEach((action) => {
                if (action.type !== ActionButtonType.BUTTON) {
                    return;
                }
                if (this.isAddOptionAction(action) && !this.hasPermissionForAddOption(metadata)) {
                    // skip to add the button for addOption if the user doesn't have permission for adding options
                    return;
                }
                if (this.isPollManagementAction(action) && !metadata.can_manage_poll) {
                    return;
                }
                content.push(
                    <ActionButton
                        key={action.id}
                        action={action}
                        postId={this.props.post.id}
                        hasVoted={this.hasVoted(action, metadata)}
                    />,
                );
            });

        return (
            <div>
                <div className='attachment-actions'>
                    {content}
                </div>
            </div>
        );
    }
}
