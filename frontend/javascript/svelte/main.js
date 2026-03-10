import { toast } from "wc-toast";

window.successToast = function (text, timeToShow = 1000) {
  toast.success(text, { duration: timeToShow });
};

window.errorToast = function (text, timeToShow = 1000) {
  toast.error(text, { duration: timeToShow });
};
