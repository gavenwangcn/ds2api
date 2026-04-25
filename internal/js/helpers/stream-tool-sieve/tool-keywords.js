'use strict';

const XML_TOOL_SEGMENT_TAGS = [
  '<tools>', '<tools\n', '<tools ', '<tool_call>', '<tool_call\n', '<tool_call ',
];

const XML_TOOL_OPENING_TAGS = [
  '<tools', '<tool_call',
];

const XML_TOOL_CLOSING_TAGS = [
  '</tools>', '</tool_call>',
];

module.exports = {
  XML_TOOL_SEGMENT_TAGS,
  XML_TOOL_OPENING_TAGS,
  XML_TOOL_CLOSING_TAGS,
};
