import React from 'react';
import PropTypes from 'prop-types';

import {ActionButtonType} from 'utils/constants';

import ActionButton from './action_button';

export default class ActionView extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        attachment: PropTypes.object.isRequired,
        pollMetadata: PropTypes.object,
        siteUrl: PropTypes.string.isRequired,

        actions: PropTypes.shape({
            fetchPollMetadata: PropTypes.func.isRequired,
        }).isRequired,
    }

    componentDidMount() {
        this.props.actions.fetchPollMetadata(this.props.siteUrl, this.props.post.props.poll_id);
    }

    /**
     * return true if the user has permission for adding option. if not, return false.
     * In details, return true in the following cases
     * - '--public-add-option' is set
     * or
     * - '--public-add-option' is NOT set AND has admin permission
     * @param {object} metadata metadata for poll
     * @return {boolean} which or not the button for add option display
     */
    hasPermissionForAddOption(metadata) {
        if (!metadata) {
            return false;
        }
        if (metadata.setting_public_add_option === true) {
            return true;
        }
        return metadata.admin_permission;
    }

    /**
     * return true if the user has already voted the option named by `name`.
     * @param {string} name
     * @param {object} metadata metadata for poll
     * @return {boolean} voted or not
     */
    hasVoted(action, metadata) {
        if (this.isAddOptionAction(action) || !metadata.voted_answers) {
            return false;
        }
        return metadata.voted_answers.indexOf(action.name) >= 0;
    }

    isAddOptionAction(action) {
        return action && (action.id === 'addOption');
    }

    render() {
        const actions = this.props.attachment.actions;
        if (!actions || !actions.length) {
            return '';
        }

        const content = [];
        const adminContent = [];
        const metadataMap = this.props.pollMetadata || {};
        const metadata = metadataMap[this.props.post.props.poll_id] || {};

        actions.
            filter((action) => action.id && action.name).
            forEach((action) => {
                switch (action.type) {
                case ActionButtonType.BUTTON:
                    if (this.isAddOptionAction(action) && !this.hasPermissionForAddOption(metadata)) {
                        // skip to add the button for addOption if the user doesn't have permission for adding options
                        break;
                    }
                    content.push(
                        <ActionButton
                            key={action.id}
                            action={action}
                            postId={this.props.post.id}
                            hasVoted={this.hasVoted(action, metadata)}
                        />
                    );
                    break;
                case ActionButtonType.MATTERPOLL_ADMIN_BUTTON:
                    if (metadata.admin_permission) {
                        adminContent.push(
                            <ActionButton
                                key={action.id}
                                action={action}
                                postId={this.props.post.id}
                                hasVoted={false}
                            />
                        );
                    }
                    break;
                default:
                    break;
                }
            });

        return (
            <div>
                <div className='attachment-actions'>
                    {content}
                </div>
                <div className='attachment-actions'>
                    {adminContent}
                </div>
            </div>
        );
    }
}
